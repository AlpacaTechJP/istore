package istore

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/gregjones/httpcache"
	"github.com/syndtr/goleveldb/leveldb"
	levelutil "github.com/syndtr/goleveldb/leveldb/util"
	"github.com/tinylib/msgp/msgp"
)

const _PathIdSeq = "sys.seq"
const _PathSeqNS = "sys.ns.seq"

type Server struct {
	Client    *http.Client
	Cache     httpcache.Cache
	Db        *leveldb.DB
	idseq     ItemId
	idseqLock sync.RWMutex
}

func copyHeader(w http.ResponseWriter, r *http.Response, header string) {
	key := http.CanonicalHeaderKey(header)
	if value, ok := r.Header[key]; ok {
		w.Header()[key] = value
	}
}

func extractTargetURL(path string) string {
	r := regexp.MustCompile("^(.*?/)([0-9a-z]+\\://.+)$")
	strs := r.FindStringSubmatch(path)

	if len(strs) <= 2 {
		return ""
	}

	dir, Url := strs[1], strs[2]

	// resolve relative path
	if strings.HasPrefix(Url, "self://") {
		path := Url[len("self://"):]
		if strings.HasPrefix(path, "./") {
			path = path[2:]
		}
		newpath := path
		if !strings.HasPrefix(path, "/") {
			newpath = dir + path
		}
		Url = "self://" + newpath
	}

	return Url
}

func NewServer(dbfile string) *Server {
	cache := httpcache.NewMemoryCache()
	cacheTransport := httpcache.NewTransport(cache)
	db, err := leveldb.OpenFile(dbfile, nil)
	if err != nil {
		glog.Error(err)
	}

	// the latest id sequence
	idseq, err := db.Get([]byte(_PathIdSeq), nil)
	if err == leveldb.ErrNotFound {
		idseq = ItemId(1).Bytes()
	}

	s := &Server{
		Client: cacheTransport.Client(),
		Cache:  cache,
		Db:     db,
		idseq:  ToItemId(idseq),
	}
	cacheTransport.Transport = s

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	glog.Infof("%s %s %s", r.Method, r.URL, r.Proto)
	switch r.Method {
	case "POST", "PUT":
		s.ServePost(w, r)
	case "DELETE":
		s.ServeDelete(w, r)
	case "GET", "HEAD":
		s.ServeGet(w, r)
	default:
		msg := fmt.Sprintf("Not implemented method %s", r.Method)
		glog.Error(msg)
		http.Error(w, msg, http.StatusNotImplemented)
	}
}

func (s *Server) NextItemId() ItemId {
	// TODO: it could be achieved by sync/atomic instead of lock
	s.idseqLock.Lock()
	defer s.idseqLock.Unlock()

	for {
		val := s.idseq
		s.idseq++
		if s.idseq == 0 {
			panic("_id wrap around")
		}
		if has, err := s.Db.Has(val.Key(), nil); err != nil {
			panic(err)
		} else if !has {
			return val
		}
	}
}

func (s *Server) PutObject(key []byte, value string, batch *leveldb.Batch, overwrite bool) (
	metabytes []byte, isnew bool, err error) {

	meta := ItemMeta{}
	// fetch item from db if exists
	if data, err := s.Db.Get(key, nil); err == nil {
		if _, err = meta.UnmarshalMsg(data); err != nil {
			glog.Error("failed to parse msgpack from db ", err)
			// continue anyway as new item
		}
	}

	// allocate id if it's new
	isnew = meta.ItemId == 0
	if isnew {
		meta.ItemId = s.NextItemId()
	}

	usermeta := map[string]interface{}{}
	if value != "" {
		// PUT completely replaces metadata, whereas POST overwrites to
		// the existing object.
		if overwrite && !isnew && meta.MetaData != nil {
			usermeta = meta.MetaData
		}
		if err := json.Unmarshal([]byte(value), &usermeta); err != nil {
			return nil, false, err
		}
	}

	meta.MetaData = usermeta

	metabytes = []byte{}
	metabytes, err = msgp.AppendIntf(metabytes, &meta)
	if err != nil {
		return nil, false, err
	}

	// User path -> metadata
	batch.Put([]byte(key), metabytes)

	if isnew {
		itemId := meta.ItemId
		// ItemId -> User path
		batch.Put(itemId.Key(), []byte(key))
		// Update the sequence number.  This could be in race condition if
		// concurrent writer updates this at the same time, and it can go
		// backward in case of restart (as far as the system is up,
		// server.idseq never goes back).  The truth is kept in the
		// ItemId -> User path and the rule of id assignment is to look at this
		// ItemId key exclusively (see NextItemId()), so the uniqueness is
		// guaranteed by this ItemId key.  That means this sequence persistency
		// is nothing but a hint to quickly catch up to the latest value.
		batch.Put([]byte(_PathIdSeq), itemId.Bytes())
	}

	return metabytes, isnew, err
}

func (s *Server) ServePost(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path
	if strings.HasSuffix(key, "/_search") {
		s.PerformSearch(w, r)
		return
	} else if strings.HasSuffix(key, "/_create_index") {
		s.CreateIndex(w, r)
		return
	} else if strings.HasSuffix(key, "/_expand") {
		s.Expand(w, r)
		return
	}

	// read user input metadata
	value := r.FormValue("metadata")
	batch := new(leveldb.Batch)
	overwrite := r.Method == "POST"
	glog.Info("about PutObject key = ", key)
	metabytes, isnew, err := s.PutObject([]byte(key), value, batch, overwrite)
	if err != nil {
		glog.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	if err := s.Db.Write(batch, nil); err != nil {
		msg := fmt.Sprintf("put failed for %s: %v", key, err)
		glog.Error(msg)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	if isnew {
		w.WriteHeader(http.StatusCreated)
	} else {
		// TODO: really?
		w.WriteHeader(http.StatusOK)
	}
	msgp.UnmarshalAsJSON(w, metabytes)
}

func (s *Server) ServeDelete(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasSuffix(path, "/") {
		iter := s.Db.NewIterator(levelutil.BytesPrefix([]byte(path)), nil)
		for iter.Next() {
			if err := s.Db.Delete(iter.Key(), nil); err != nil {
				glog.Error(err)
				// keep going...
			}
		}
	} else {
		err := s.Db.Delete([]byte(path), nil)

		if err == leveldb.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		// TODO: delete ItemId -> path
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) ServeList(w http.ResponseWriter, r *http.Request, path string) {
	iter := s.Db.NewIterator(levelutil.BytesPrefix([]byte(path)), nil)
	results := []interface{}{}
	for iter.Next() {
		meta := ItemMeta{}

		if path == _PathSeqNS {
			meta.ItemId = ToItemId(iter.Key()[len(_PathSeqNS):])
			meta.FilePath = string(iter.Value())
		} else {
			value := iter.Value()
			if value != nil {
				if _, err := meta.UnmarshalMsg(value); err != nil {
					glog.Error("failed to unmarshal metadata from db ", err)
				}
			}
			meta.FilePath = string(iter.Key())
		}
		results = append(results, meta)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		msg := fmt.Sprint(err)
		glog.Error(msg)
		http.Error(w, "Error", http.StatusInternalServerError)
	}

	w.Header()["Content-type"] = []string{"application/json"}
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(results); err != nil {
		glog.Error(err)
	}
}

func (s *Server) ServeGet(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.HasSuffix(path, "/") {
		s.ServeList(w, r, path)
		return
	} else if path == "/"+_PathSeqNS {
		s.ServeList(w, r, _PathSeqNS)
		return
	}

	if _, err := s.Db.Get([]byte(path), nil); err != nil {
		if err == leveldb.ErrNotFound {
			glog.Error(path, " not found")
			http.NotFound(w, r)
			return
		}
		msg := fmt.Sprintf("error while reading %s: %v", path, err)
		glog.Error(msg)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	resp, err := s.GetApply(r)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if resp == nil {
		} else {
			statusCode = resp.StatusCode
		}
		glog.Error(err, statusCode)
		http.Error(w, "Error", statusCode)
		return
	}

	copyHeader(w, resp, "Last-Modified")
	copyHeader(w, resp, "Expires")
	copyHeader(w, resp, "Etag")
	copyHeader(w, resp, "Content-Length")
	copyHeader(w, resp, "Content-Type")
	io.Copy(w, resp.Body)
}

func (s *Server) GetApply(r *http.Request) (*http.Response, error) {
	path := r.URL.Path

	Url := extractTargetURL(path)
	if Url == "" {
		// TODO: return NotFound?
		return nil, fmt.Errorf("target not found in path %s", path)
	}

	resp, err := s.Client.Get(Url)
	if err != nil {
		if resp != nil {
			return resp, fmt.Errorf("remote URL %q returned status: %v\n%v", Url, resp.Status, err)
		}
		return resp, err
	}

	return handleApply(resp, r)
}

func handleApply(resp *http.Response, r *http.Request) (newresp *http.Response, err error) {
	apply := r.FormValue("apply")

	var img []byte
	switch apply {
	case "adjustBrightness":
		percentage, err := strconv.ParseFloat(r.FormValue("percentage"), 64)
		if img, err = adjustBrightness(resp.Body, percentage); err != nil {
			return nil, err
		}

	case "adjustContrast":
		percentage, err := strconv.ParseFloat(r.FormValue("percentage"), 64)
		if img, err = adjustContrast(resp.Body, percentage); err != nil {
			return nil, err
		}

	case "adjustGamma":
		gamma, err := strconv.ParseFloat(r.FormValue("gamma"), 64)
		if img, err = adjustGamma(resp.Body, gamma); err != nil {
			return nil, err
		}

	case "adjustSigmoid":
		midpoint, err := strconv.ParseFloat(r.FormValue("midpoint"), 64)
		factor, err := strconv.ParseFloat(r.FormValue("factor"), 64)
		if img, err = adjustSigmoid(resp.Body, midpoint, factor); err != nil {
			return nil, err
		}

	case "blur":
		sigma, _ := strconv.ParseFloat(r.FormValue("sigma"), 64)
		if img, err = blur(resp.Body, sigma); err != nil {
			return nil, err
		}

	case "crop":
		x0, err := strconv.Atoi(r.FormValue("x0"))
		y0, err := strconv.Atoi(r.FormValue("y0"))
		x1, err := strconv.Atoi(r.FormValue("x1"))
		y1, err := strconv.Atoi(r.FormValue("y1"))
		if x0 == 0 && y0 == 0 && x1 == 0 && y1 == 0 {
			return resp, nil
		}
		if img, err = crop(resp.Body, x0, y0, x1, y1); err != nil {
			return nil, err
		}

	case "fit":
		w, err := strconv.Atoi(r.FormValue("w"))
		h, err := strconv.Atoi(r.FormValue("h"))
		if img, err = fit(resp.Body, w, h); err != nil {
			return nil, err
		}

	case "flipH":
		if img, err = flipH(resp.Body); err != nil {
			return nil, err
		}

	case "flipV":
		if img, err = flipV(resp.Body); err != nil {
			return nil, err
		}

	case "grayscale":
		if img, err = grayscale(resp.Body); err != nil {
			return nil, err
		}

	case "invert":
		if img, err = invert(resp.Body); err != nil {
			return nil, err
		}

	case "sharpen":
		sigmoid, err := strconv.ParseFloat(r.FormValue("sigmoid"), 64)
		if img, err = sharpen(resp.Body, sigmoid); err != nil {
			return nil, err
		}

	case "transpose":
		if img, err = transpose(resp.Body); err != nil {
			return nil, err
		}

	case "transverse":
		if img, err = transverse(resp.Body); err != nil {
			return nil, err
		}

	case "resize":
		w, err := strconv.Atoi(r.FormValue("w"))
		h, err := strconv.Atoi(r.FormValue("h"))
		if w == 0 && h == 0 {
			return resp, nil
		}
		if img, err = resize(resp.Body, w, h); err != nil {
			return nil, err
		}

	case "frame":
		sec, err := strconv.Atoi(r.FormValue("sec"))
		if img, err = frame(resp.Body, sec); err != nil {
			return nil, err
		}

		buf := new(bytes.Buffer)
		fmt.Fprintf(buf, "%s %s", resp.Proto, resp.Status)
		fmt.Fprintf(buf, "Content-Length: %d\n", len(img))
		fmt.Fprintf(buf, "Content-type: image/jpeg\n\n")
		buf.Write(img)

		return http.ReadResponse(bufio.NewReader(buf), r)

	default:
		return resp, nil
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s %s", resp.Proto, resp.Status)
	resp.Header.WriteSubset(buf, map[string]bool{"Content-Length": true})
	fmt.Fprintf(buf, "Content-Length: %d\n\n", len(img))
	buf.Write(img)

	return http.ReadResponse(bufio.NewReader(buf), r)
}

// -----
// some thoughts
// curl -X POST http://localhost:9999/mybucket/events/19/_search -d '
// {
//   "similar": {
//     "to": "self:///mybucket/events/19/foobar.jpg",
//     "by": "feature"
//   }
// }
//
// curl -X POST http://localhost:9999/mybucket/events/19/_create_index -d '
// {
//   "similar": {
//     "by": "feature"
//   }
// }

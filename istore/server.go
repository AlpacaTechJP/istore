package istore

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/golang/glog"
	"github.com/gregjones/httpcache"
	"github.com/syndtr/goleveldb/leveldb"
)

const _DbFile = "/tmp/metadb"

type Server struct {
	Client *http.Client
	Cache  httpcache.Cache
	Db     *leveldb.DB
}

func copyHeader(w http.ResponseWriter, r *http.Response, header string) {
	key := http.CanonicalHeaderKey(header)
	if value, ok := r.Header[key]; ok {
		w.Header()[key] = value
	}
}

func NewServer() *Server {
	cache := httpcache.NewMemoryCache()
	client := &http.Client{}
	client.Transport = httpcache.NewTransport(cache)
	db, err := leveldb.OpenFile(_DbFile, nil)
	if err != nil {
		glog.Error(err)
	}

	return &Server{
		Client: client,
		Cache:  cache,
		Db:     db,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	glog.Infof("%v", r)
	switch r.Method {
	case "POST", "PUT":
		s.ServePost(w, r)
	case "GET", "HEAD":
		s.ServeGet(w, r)
	default:
		msg := fmt.Sprintf("Not implemented method %s", r.Method)
		glog.Error(msg)
		http.Error(w, msg, http.StatusNotImplemented)
	}
}

func (s *Server) ServePost(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path
	meta := r.FormValue("meta-data")

	if err := s.Db.Put([]byte(key), []byte(meta), nil); err != nil {
		msg := fmt.Sprintf("put failed for %s: %v", key, err)
		glog.Error(msg)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("OK"))
}

func extractTargetURL(path string) string {
	r := regexp.MustCompile("^.+/([0-9a-z]+\\://.+)$")
	strs := r.FindStringSubmatch(path)

	if len(strs) > 1 {
		return strs[1]
	}
	return ""
}

func (s *Server) ServeGet(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if _, err := s.Db.Get([]byte(path), nil); err != nil {
		if err == leveldb.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		msg := fmt.Sprintf("error while reading %s: %v", path, err)
		glog.Error(msg)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	urlstr := extractTargetURL(path)
	if urlstr == "" {
		msg := fmt.Sprintf("target not found in path %s", path)
		glog.Info(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	client := s.Client
	resp, err := client.Get(urlstr)
	if err != nil {
		var msg string
		statusCode := http.StatusBadRequest
		if resp == nil {
			msg = fmt.Sprintf("%v", err)
		} else {
			msg = fmt.Sprintf("remote URL %q returned status: %v", urlstr, resp.Status)
			statusCode = resp.StatusCode
		}
		glog.Error(msg)
		http.Error(w, msg, statusCode)
		return
	}

	copyHeader(w, resp, "Last-Modified")
	copyHeader(w, resp, "Expires")
	copyHeader(w, resp, "Etag")
	copyHeader(w, resp, "Content-Length")
	copyHeader(w, resp, "Content-Type")
	io.Copy(w, resp.Body)
}

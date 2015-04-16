package istore

import (
	"bytes"
	"encoding/gob"
	"sort"
)

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/AlpacaDB/istore/lsh"
	"github.com/golang/glog"
	levelutil "github.com/syndtr/goleveldb/leveldb/util"
)

type Similarity struct {
	To    string `json:"to,omitempty"`
	By    string `json:"by,omitempty"`
	Limit int    `json:"limit,omitempty"`
	to    ItemMeta
}

type Query struct {
	Similar Similarity `json:"similar,omitempty"`
	key     string
}

func jsonArrayToFloat32(data interface{}) []float32 {
	// TODO: sigh, []interface{} -> []float32
	if vec, ok := data.([]interface{}); ok {
		nv := make([]float32, len(vec), len(vec))
		for i, val := range vec {
			if d, ok := val.(float64); ok {
				nv[i] = float32(d)
			}
		}
		return nv
	}

	return nil
}

func convertJsonForQuery(value []byte, by string, item *ItemMeta) bool {
	if value != nil {
		if err := json.Unmarshal(value, &item); err != nil {
			glog.Error("failed to unmarshal metadata from db", err)
			return false
		}
	}

	if v, ok := item.MetaData[by]; !ok {
		return false
	} else {
		slice := jsonArrayToFloat32(v)
		if slice == nil {
			return false
		}
		item.MetaData[by] = slice
	}

	return true
}

type ItemGetter struct {
	server *Server
	query  *Query
}
type ItemVector struct {
	item  ItemMeta
	query *Query
}

func (g *ItemGetter) Get(itemid uint64) lsh.Item {
	if key, err := g.server.Db.Get(ItemId(itemid).Key(), nil); err == nil {
		if data, err := g.server.Db.Get(key, nil); err == nil {
			item := ItemMeta{}
			item.FilePath = string(key)
			convertJsonForQuery(data, g.query.Similar.By, &item)
			return &ItemVector{
				item:  item,
				query: g.query,
			}
		} else {
			glog.Error("could not find ", string(key), err)
		}
	} else {
		glog.Error("could not find itemid ", itemid, err)
	}
	return nil
}

func (v *ItemVector) Vector() []float32 {
	return v.item.MetaData[v.query.Similar.By].([]float32)
}

func (s *Server) PerformSearchIndex(query *Query, index *lsh.Indexer) []ItemMeta {
	itemGetter := &ItemGetter{
		server: s,
		query:  query,
	}
	glog.Info(query.Similar.to)
	glog.Info(query.Similar.to.MetaData)
	vec_to := query.Similar.to.MetaData[query.Similar.By].([]float32)
	results := index.Search(vec_to, query.Similar.Limit, itemGetter)
	items := make([]ItemMeta, len(results), len(results))
	for i, v := range results {
		items[i] = v.(*ItemVector).item
	}
	return items
}

func (s *Server) PerformSearchBluteForce(query *Query) []ItemMeta {
	iter := s.Db.NewIterator(levelutil.BytesPrefix([]byte(query.key)), nil)
	defer iter.Release()
	items := make([]ItemMeta, 0)
	for iter.Next() {
		item := ItemMeta{}
		item.FilePath = string(iter.Key())

		value := iter.Value()
		if !convertJsonForQuery(value, query.Similar.By, &item) {
			continue
		}
		items = append(items, item)
	}

	angular := lsh.Angular{}
	(&itemSort{
		LenFunc: func() int {
			return len(items)
		},
		SwapFunc: func(i, j int) {
			items[i], items[j] = items[j], items[i]
		},
		LessFunc: func(i, j int) bool {
			// they must appear as converted above.
			vec_i := items[i].MetaData[query.Similar.By].([]float32)
			vec_j := items[j].MetaData[query.Similar.By].([]float32)
			vec_to := query.Similar.to.MetaData[query.Similar.By].([]float32)
			// TODO: check dimension
			dist_i := angular.Distance(vec_i, vec_to)
			dist_j := angular.Distance(vec_j, vec_to)
			return dist_i < dist_j
		},
	}).Sort()

	limit := query.Similar.Limit
	if len(items) < limit {
		limit = len(items)
	}

	return items[0:limit]
}

func (s *Server) PerformSearch(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path
	// key has suffix "/_search"
	key = key[0 : len(key)-len("_search")]

	if !strings.HasPrefix(key, "/") {
		http.Error(w, "search key should finish with '/'", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	query := Query{
		key: key,
	}
	if err := decoder.Decode(&query); err != nil {
		http.Error(w, "unrecognizable query", http.StatusBadRequest)
		return
	}

	to_data, err := s.Db.Get([]byte(query.Similar.To), nil)
	if err != nil {
		glog.Error(err)
		http.Error(w, "similar.to must be present", http.StatusBadRequest)
		return
	}
	if !convertJsonForQuery(to_data, query.Similar.By, &query.Similar.to) {
		http.Error(w, "similar.to item is not usable for query", http.StatusBadRequest)
		return
	}

	var items []ItemMeta
	if index_data, err := s.Db.Get([]byte(key+"_index"), nil); err == nil {
		index := new(lsh.Indexer)
		decoder := gob.NewDecoder(bytes.NewBuffer(index_data))
		index.Decode(decoder)
		items = s.PerformSearchIndex(&query, index)
	} else {
		items = s.PerformSearchBluteForce(&query)
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(items); err != nil {
		glog.Error("failed to write search result", err)
	}
}

type itemSort struct {
	LenFunc  func() int
	SwapFunc func(int, int)
	LessFunc func(int, int) bool
}

func (s *itemSort) Len() int {
	return s.LenFunc()
}
func (s *itemSort) Swap(i, j int) {
	s.SwapFunc(i, j)
}
func (s *itemSort) Less(i, j int) bool {
	return s.LessFunc(i, j)
}
func (s *itemSort) Sort() {
	sort.Sort(s)
}

func (s *Server) CreateIndex(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path
	// suffix _create_index
	key = key[0 : len(key)-len("_create_index")]

	if !strings.HasSuffix(key, "/") {
		http.Error(w, "create index key should finish with '/'", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	query := Query{}
	if err := decoder.Decode(&query); err != nil {
		http.Error(w, "unrecognized query", http.StatusBadRequest)
		return
	}

	var index *lsh.Indexer
	iter := s.Db.NewIterator(levelutil.BytesPrefix([]byte(key)), nil)
	defer iter.Release()
	for iter.Next() {
		item := ItemMeta{}

		value := iter.Value()
		if !convertJsonForQuery(value, query.Similar.By, &item) {
			continue
		}
		vec := item.MetaData[query.Similar.By].([]float32)
		if index == nil {
			index = lsh.NewIndexer(0, 8, len(vec))
		}
		index.Add(uint64(item.ItemId), vec)
	}

	if index == nil {
		http.NotFound(w, r)
		return
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	index.Encode(encoder)
	if err := s.Db.Put([]byte(key+"_index"), buf.Bytes(), nil); err != nil {
		glog.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

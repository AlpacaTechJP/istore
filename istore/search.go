package istore

import (
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
}

type Query struct {
	Similar Similarity `json:"similar,omitempty"`
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

func (s *Server) PerformSearch(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path
	// key has suffix "/_search"
	key = key[0 : len(key)-len("_search")]

	if !strings.HasPrefix(key, "/") {
		http.Error(w, "search key should finish with '/'", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	query := Query{}
	if err := decoder.Decode(&query); err != nil {
		http.Error(w, "unrecognizable query", http.StatusBadRequest)
		return
	}

	iter := s.Db.NewIterator(levelutil.BytesPrefix([]byte(key)), nil)
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

	limit := query.Similar.Limit
	if len(items) < limit {
		limit = len(items)
	}

	to_data, err := s.Db.Get([]byte(query.Similar.To), nil)
	if err != nil {
		glog.Error(err)
		http.Error(w, "similar.to must be present", http.StatusBadRequest)
		return
	}

	to := ItemMeta{}
	if !convertJsonForQuery(to_data, query.Similar.By, &to) {
		http.Error(w, "similar.to item is not usable for query", http.StatusBadRequest)
		return
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
			vec_to := to.MetaData[query.Similar.By].([]float32)
			// TODO: check dimension
			dist_i := angular.Distance(vec_i, vec_to)
			dist_j := angular.Distance(vec_j, vec_to)
			return dist_i < dist_j
		},
	}).Sort()

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(items[0:limit]); err != nil {
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

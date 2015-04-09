package lsh

import (
	"bytes"
	"fmt"
	"math"
	"sort"

	"github.com/AlpacaDB/istore/bitvector"
)

type IndexStats struct {
	Splitters      [][]float32
	Buckets        []*IndexBucketStats
	NumItems       int
	NumKeys        int
	NumItemsAvg    float64
	NumItemsStddev float64
}

type IndexBucketStats struct {
	BitKey      *bitvector.BitVector
	NumItems    int
	PageNumbers []int
}

func (idx *Indexer) Stats() *IndexStats {
	splitters := make([][]float32, len(idx.hyperplane), len(idx.hyperplane))
	for i, h := range idx.hyperplane {
		splitters[i] = append(splitters[i], h...)
	}
	keys := []int{}
	for k, _ := range idx.lookup {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	var sum, squaresum float64
	buckets := make([]*IndexBucketStats, len(keys), len(keys))
	for i, k := range keys {
		pageno := idx.lookup[uint32(k)]
		bv := bitvector.FromUint32(uint32(k), idx.bitsize)
		iter := idx.storage.pageIterator(pageno)
		var nitems = 0
		pagenolist := []int{}
		for iter.next() {
			page := iter.page()
			nitems += page.CountItems()
			pagenolist = append(pagenolist, iter.pageno())
		}
		buckets[i] = &IndexBucketStats{
			BitKey:      bv,
			NumItems:    nitems,
			PageNumbers: pagenolist,
		}
		sum += float64(nitems)
		squaresum += float64(nitems) * float64(nitems)
	}
	mean := sum / float64(len(keys))
	stddev := math.Sqrt(squaresum/float64(len(keys)) - mean*mean)

	return &IndexStats{
		Splitters:      splitters,
		Buckets:        buckets,
		NumItems:       int(sum),
		NumKeys:        len(buckets),
		NumItemsAvg:    mean,
		NumItemsStddev: stddev,
	}
}

func (stats *IndexStats) Dump() string {
	buffer := new(bytes.Buffer)

	// buffer.WriteString("splitter --- \n")
	// for i, h := range stats.Splitters {
	// 	buffer.WriteString(fmt.Sprintf("%d %v\n", i, h))
	// }

	for _, bucket := range stats.Buckets {
		bv := bucket.BitKey
		buffer.WriteString(fmt.Sprintf("key(%08d:%s) -> page(%v) = %d items\n",
			bv.Uint32(), bv.String(), bucket.PageNumbers, bucket.NumItems))
	}
	buffer.WriteString(fmt.Sprintf(
		"total items = %d / keys = %d, mean = %f, stddev = %f",
		stats.NumItems, stats.NumKeys, stats.NumItemsAvg, stats.NumItemsStddev))

	return buffer.String()
}

type Encoder interface {
	Encode(interface{}) error
}

type Decoder interface {
	Decode(interface{}) error
}

func (idx *Indexer) Encode(enc Encoder) error {
	// TODO: set version code
	enc.Encode(idx.seed)
	enc.Encode(idx.bitsize)
	enc.Encode(idx.vecsize)
	switch idx.distance.(type) {
	case Angular:
		enc.Encode("angular")
	}
	enc.Encode(idx.hyperplane)
	enc.Encode(idx.lookup)

	// TODO: a lot of optimization...
	enc.Encode(len(idx.storage.pages))
	for _, p := range idx.storage.pages {
		enc.Encode(p.nitems)
		enc.Encode(p.link)
		enc.Encode(p.items)
	}

	return nil
}

func (idx *Indexer) Decode(dec Decoder) error {
	dec.Decode(&idx.seed)
	dec.Decode(&idx.bitsize)
	dec.Decode(&idx.vecsize)
	var distance string
	dec.Decode(&distance)
	switch distance {
	case "angular":
		idx.distance = Angular{}
	}
	dec.Decode(&idx.hyperplane)
	dec.Decode(&idx.lookup)

	var npages int
	dec.Decode(&npages)
	if idx.storage == nil {
		idx.storage = &Storage{}
	}
	idx.storage.pages = make([]Page, npages, npages)
	for i := 0; i < npages; i++ {
		dec.Decode(&idx.storage.pages[i].nitems)
		dec.Decode(&idx.storage.pages[i].link)
		dec.Decode(&idx.storage.pages[i].items)
	}

	return nil
}

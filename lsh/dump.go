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

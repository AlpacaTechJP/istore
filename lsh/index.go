package lsh

import (
	"bytes"
	"fmt"
	"github.com/AlpacaDB/istore/bitvector"
	"math"
	"sort"
)

type Indexer struct {
	seed       int64
	bitsize    int
	vecsize    int
	distance   Distance
	hyperplane [][]float32
	storage    *Storage
	lookup     map[uint32]int
}

func NewIndexer(seed int64, bitsize int, vecsize int) *Indexer {
	if bitsize > 32 {
		panic("currently bitsize > 32 is not supported")
	}
	idx := &Indexer{
		seed:     seed,
		bitsize:  bitsize,
		vecsize:  vecsize,
		distance: Angular{},
		storage:  &Storage{},
		lookup:   map[uint32]int{},
	}

	// init hyperplane
	generator := NewRandomVectorGen(seed, vecsize)
	idx.hyperplane = make([][]float32, bitsize, bitsize)
	for i := 0; i < bitsize; i++ {
		idx.hyperplane[i] = generator.Next()
	}

	return idx
}

func (idx *Indexer) Add(itemid uint64, vec []float32) {
	key := idx.distance.GetBitVector(idx.hyperplane, vec)
	pageno, ok := idx.lookup[key.Uint32()]
	if !ok {
		pageno = idx.storage.allocatePage()
		idx.lookup[key.Uint32()] = pageno
	}
	idx.storage.Add(itemid, pageno)
}

// mainly for debug and analysis
func (idx *Indexer) GetBitVector(vec []float32) *bitvector.BitVector {
	return idx.distance.GetBitVector(idx.hyperplane, vec)
}

// Search searches items close to the given vector up to the limit.
// Currently this returns more than limits by looking at the bitvectors
// with the same distance, without desired order.  The caller should
// recall the vector and re-order by the metrics.  We will probably
// want another interface that does this work.
func (idx *Indexer) Search(vec []float32, limit int) []uint64 {
	key := idx.distance.GetBitVector(idx.hyperplane, vec)

	lkeys := make([]*bitvector.BitVector, 0, len(idx.lookup))
	for k, _ := range idx.lookup {
		lkeys = append(lkeys, bitvector.FromUint32(k, idx.bitsize))
	}
	bitvector.Sort(lkeys).From(key)

	items := make([]uint64, 0, limit)
	var lastdist int
	for len(lkeys) > 0 {
		thiskey := lkeys[0]
		thisdist := bitvector.Hamming(key, thiskey)

		// We continue to collect items even if it exeeds requested limit,
		// as far as the haming distance is the same.
		if lastdist != thisdist && len(items) >= limit {
			break
		}
		lastdist = thisdist
		lkeys = lkeys[1:]

		// the key should exist
		pageno := idx.lookup[thiskey.Uint32()]
		iter := idx.storage.pageIterator(pageno)
		for iter.next() {
			page := iter.page()
			items = append(items, page.Gets()...)
		}
	}

	return items
}

func (idx *Indexer) Dump() string {
	buffer := new(bytes.Buffer)

	buffer.WriteString("hyperplane --- \n")
	for i, h := range idx.hyperplane {
		buffer.WriteString(fmt.Sprintf("%d: %v\n", i, h))
	}

	keys := make([]int, 0)
	for k, _ := range idx.lookup {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	var sum, squaresum float64
	for _, k := range keys {
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
		buffer.WriteString(fmt.Sprintf("key(%08d:%s) -> page(%v) = %d items\n",
			k, bv.String(), pagenolist, nitems))

		sum += float64(nitems)
		squaresum += float64(nitems) * float64(nitems)
	}
	mean := sum / float64(len(keys))
	stddev := math.Sqrt(squaresum/float64(len(keys)) - mean*mean)
	buffer.WriteString(fmt.Sprintf("total items = %d / keys = %d, mean = %f, stddev = %f", int(sum), len(keys), mean, stddev))

	return buffer.String()
}

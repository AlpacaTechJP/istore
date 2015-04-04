package lsh

import (
	"math"
	"math/rand"

	"github.com/AlpacaDB/istore/bitvector"
)

type Indexer struct {
	rng        *rand.Rand
	seed       int64
	bitsize    int
	vecsize    int
	distance   Distance
	hyperplane [][]float32
	storage    *Storage
	lookup     map[uint32]int
}

type Storage struct {
	hash  []int32
	pages []Page
}

type Page [1024]uint64

func NewIndexer(seed int64, bitsize int, vecsize int) *Indexer {
	if bitsize > 32 {
		panic("currently bitsize > 32 is not supported")
	}
	idx := &Indexer{
		rng:      rand.New(rand.NewSource(seed)),
		seed:     seed,
		bitsize:  bitsize,
		vecsize:  vecsize,
		distance: Angular{},
		storage:  &Storage{},
		lookup:   map[uint32]int{},
	}

	// init hyperplane
	idx.hyperplane = make([][]float32, bitsize, bitsize)
	for i := 0; i < bitsize; i++ {
		vector := make([]float32, vecsize, vecsize)
		var sum float64 = 0
		for j := 0; j < vecsize; j++ {
			vector[j] = float32(idx.rng.NormFloat64())
			sum += float64(vector[j]) * float64(vector[j])
		}
		norm := float32(math.Sqrt(sum))
		// normalize
		for j := 0; j < vecsize; j++ {
			vector[j] /= norm
		}
		idx.hyperplane[i] = vector
	}

	return idx
}

func (idx *Indexer) Add(itemid uint64, vec []float32) {
	key := idx.distance.GetBitVector(idx.hyperplane, vec)
	pageno, ok := idx.lookup[key.Uint32()]
	if !ok {
		pageno = idx.allocatePage()
		idx.lookup[key.Uint32()] = pageno
	}
	idx.storage.pages[pageno].Add(itemid)
}

func (idx *Indexer) allocatePage() int {
	n := len(idx.storage.pages)
	idx.storage.pages = idx.storage.pages[0 : n+1]
	return n
}

// Search searches items close to the given vector up to the limit.
func (idx *Indexer) Search(vec []float32, limit int) []uint64 {
	key := idx.distance.GetBitVector(idx.hyperplane, vec)

	lkeys := make([]*bitvector.BitVector, 0, len(idx.lookup))
	for k, _ := range idx.lookup {
		lkeys = append(lkeys, bitvector.FromUint32(k, idx.bitsize))
	}
	bitvector.BitVectorSlice(lkeys).SortFrom(key)

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
		page := &idx.storage.pages[pageno]
		i := 0
		for {
			if page.Get(i) == 0 {
				break
			}
			items = append(items, page.Get(i))
			i++

			// TODO: go to next page if it exceeds items in page
		}
	}

	// TODO: sort and limit
	return items
}

func (p *Page) Add(itemid uint64) {
	itemlen := (*p)[0] + 1
	(*p)[int(itemlen)] = itemid
	(*p)[0] = itemlen
}

func (p *Page) Get(n int) uint64 {
	itemlen := (*p)[0]
	if int(itemlen) <= n {
		return 0
	}
	return (*p)[n+1]
}

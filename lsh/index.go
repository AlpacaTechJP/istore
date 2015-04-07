package lsh

import (
	"sort"

	"github.com/AlpacaDB/istore/bitvector"
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

// Candidates searches items close to the given vector, roughly up to limit.
// This returns more than limits by looking at the bitvectors
// with the same distance, without desired order.  The caller should
// recall the vector and re-order by the metrics.
func (idx *Indexer) Candidates(vec []float32, limit int) []uint64 {
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

type Item interface {
	Vector() []float32
}

type ItemGetter interface {
	Get(itemid uint64) Item
}

type itemSort []Item

func (s itemSort) Len() int {
	return len(s)
}

func (s itemSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type itemSortFrom struct {
	itemSort
	cent []float32
	dist Distance
}

func (s itemSort) From(cent []float32, dist Distance) {
	sort.Sort(&itemSortFrom{s, cent, dist})
}

type SimpleRecord struct {
	itemid uint64
	vector []float32
}

// Vector implements Item.Vector()
func (r *SimpleRecord) Vector() []float32 {
	return r.vector
}

type SimpleRecords [][]float32

// Get implements ItemGetter.Get()
func (r SimpleRecords) Get(itemid uint64) Item {
	return &SimpleRecord{itemid, r[itemid-1]}
}

func (s *itemSortFrom) Less(i, j int) bool {
	dist_i := s.dist.Distance(s.itemSort[i].Vector(), s.cent)
	dist_j := s.dist.Distance(s.itemSort[j].Vector(), s.cent)
	return dist_i < dist_j
}

func (idx *Indexer) Qualify(vec []float32, limit int, getter ItemGetter, candidates []uint64) []Item {
	if len(candidates) < limit {
		limit = len(candidates)
	}
	items := make([]Item, 0, len(candidates))
	for _, itemid := range candidates {
		items = append(items, getter.Get(itemid))
	}

	itemSort(items).From(vec, idx.distance)

	return items[:limit]
}

func (idx *Indexer) Search(vec []float32, limit int, getter ItemGetter) []Item {
	candidates := idx.Candidates(vec, limit)
	return idx.Qualify(vec, limit, getter, candidates)
}

package lsh

import (
	"math"
	"math/rand"
)

type Distance interface {
	Distance(x, y []float32) float32
	GetBitVector(vecs [][]float32, v []float32) *BitVector
}

type Angular struct{}

func (_ Angular) Distance(x, y []float32) float32 {
	if len(x) != len(y) {
		panic("")
	}
	var xx, yy, xy float32 = 0, 0, 0
	for i := 0; i < len(x); i++ {
		xx += x[i] * x[i]
		yy += y[i] * y[i]
		xy += x[i] * y[i]
	}

	xxyy := xx * yy
	if xxyy > 0 {
		return float32(2.0 - 2.0*float64(xy)/math.Sqrt(float64(xxyy)))
	}
	return 2.0 // cos is 0
}

func (_ Angular) GetBitVector(vecs [][]float32, v []float32) *BitVector {
	bv := NewBitVector(len(vecs))

	return bv
}

type Euclidean struct{}

func (e *Euclidean) distance(x, y []float32) float32 {
	if len(x) != len(y) {
		panic("")
	}

	var d float32 = 0
	for i := 0; i < len(x); i++ {
		d += (x[i] - y[i]) * (x[i] - y[i])
	}
	return d
}

type Indexer struct {
	rng        *rand.Rand
	seed       int64
	bitsize    int
	vecsize    int
	distance   Distance
	hyperplane [][]float32
	storage    *Storage
	lookup     map[uint64]int
}

type Storage struct {
	hash  []int32
	pages []Page
}

type Page [1024]uint64

func NewIndexer(seed int64, bitsize int, vecsize int) *Indexer {
	idx := &Indexer{
		rng:      rand.New(rand.NewSource(seed)),
		seed:     seed,
		bitsize:  bitsize,
		vecsize:  vecsize,
		distance: Angular{},
		storage:  &Storage{},
		lookup:   map[uint64]int{},
	}

	// init hyperplane
	idx.hyperplane = make([][]float32, bitsize, bitsize)
	for i := 0; i < bitsize; i++ {
		vector := make([]float32, vecsize, vecsize)
		for j := 0; j < vecsize; j++ {
			vector[j] = float32(idx.rng.NormFloat64())
		}
		idx.hyperplane[i] = vector
	}

	return idx
}

func (idx *Indexer) Add(itemid uint64, vec []float32) {
	key := idx.distance.GetBitVector(idx.hyperplane, vec)
	pageno, ok := idx.lookup[key.Uint64()]
	if !ok {
		pageno = idx.allocatePage()
		idx.lookup[key.Uint64()] = pageno
	}
	idx.storage.pages[pageno].Add(itemid)
}

func (idx *Indexer) allocatePage() int {
	n := len(idx.storage.pages)
	idx.storage.pages = idx.storage.pages[0 : n+1]
	return n
}

// Search searches items similar to the given vector up to limit number.
// Currently it looks up only the same hash value.  (look for another
// bucket is TODO)
func (idx *Indexer) Search(vec []float32, limit int) []uint64 {
	key := idx.distance.GetBitVector(idx.hyperplane, vec)
	pageno, ok := idx.lookup[key.Uint64()]
	if !ok {
		return []uint64{}
	}
	page := &idx.storage.pages[pageno]

	result := make([]uint64, 0, limit)
	i := 0
	for {
		if len(result) == limit {
			break
		}

		if page.Get(i) == 0 {
			break
		}
		result = append(result, page.Get(i))
		i++

		// TODO: go to next page if i exceeds items in page
	}

	return result
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

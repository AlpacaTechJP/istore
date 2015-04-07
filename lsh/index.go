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

// Add adds item to one of the pages and return the pageno that
// the items belongs to.
func (s *Storage) Add(itemid uint64, pageno int) int {
	page := &s.pages[pageno]
	next := page.Next()
	for next != 0 {
		page = &s.pages[next]
		pageno = next
		next = page.Next()
	}

	if page.Full() {
		newpageno := s.allocatePage()
		page.Link(newpageno)
		page = &s.pages[newpageno]
		pageno = newpageno
	}

	page.Add(itemid)

	return pageno
}

func (s *Storage) allocatePage() int {
	n := len(s.pages)
	s.pages = append(s.pages, Page{})
	return n
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
	bitvector.Slice(lkeys).SortFrom(key)

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
		page := &idx.storage.pages[pageno]
		bv := bitvector.FromUint32(uint32(k), idx.bitsize)
		nitems := page.CountItems()
		buffer.WriteString(fmt.Sprintf("key(%08d:%s) -> page(%d) = %d items\n",
			k, bv.String(), pageno, nitems))

		sum += float64(nitems)
		squaresum += float64(nitems) * float64(nitems)
	}
	mean := sum / float64(len(keys))
	stddev := math.Sqrt(squaresum/float64(len(keys)) - mean*mean)
	buffer.WriteString(fmt.Sprintf("total items = %d / keys = %d, mean = %f, stddev = %f", int(sum), len(keys), mean, stddev))

	return buffer.String()
}

func (p *Page) Add(itemid uint64) {
	itemlen := p.incrementItems()
	(*p)[itemlen] = itemid
}

func (p *Page) Get(n int) uint64 {
	itemlen := (*p)[0]
	if int(itemlen) <= n {
		return 0
	}
	return (*p)[n+1]
}

func (p *Page) CountItems() int {
	return int((*p)[0] & 0xffffffff)
}

func (p *Page) incrementItems() int {
	itemlen := p.CountItems() + 1
	(*p)[0] = ((*p)[0] & 0xffffffff00000000) | (uint64(itemlen) & 0xffffffff)
	return itemlen

}

func (p *Page) Next() int {
	return int((*p)[0] >> 32)
}

func (p *Page) Link(next int) {
	(*p)[0] |= (uint64(next) << 32)
}

func (p *Page) Full() bool {
	// the first byte is for count/linkage
	return p.CountItems() == len(*p) - 1
}

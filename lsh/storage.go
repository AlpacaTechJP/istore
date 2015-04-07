package lsh

type Storage struct {
	hash  []int32
	pages []Page
}

type Page struct {
	nitems int32
	link   int32
	items  [1023]uint64
}

// Add adds item to one of the pages and return the pageno that
// the items belongs to.
func (s *Storage) Add(itemid uint64, pageno int) int {
	iter := s.pageIterator(pageno)
	var page *Page
	for iter.next() {
		page = iter.page()
		pageno = iter.pageno()
	}

	// Move to the new page if the current page is full.
	if page.Full() {
		newpageno := s.allocatePage()
		page.Link(newpageno)
		page = s.getPage(newpageno)
		pageno = newpageno
	}

	page.Add(itemid)

	return pageno
}

func (s *Storage) getPage(pageno int) *Page {
	return &s.pages[pageno]
}

// allocatePage appends new page at the end of array, and returns the page number of it.
func (s *Storage) allocatePage() int {
	n := len(s.pages)
	s.pages = append(s.pages, Page{})
	s.pages[n].Init()
	return n
}

// pageIter is an iterator over multiple pages that are linked.
// Use this way:
// 	iter := storage.pageIterator()
// 	for iter.next() {
// 		page := iter.page()
// 		fmt.Println(page.CountItems())
// 	}
type pageIter struct {
	storage        *Storage
	currno, nextno int
}

func (s *Storage) pageIterator(pageno int) *pageIter {
	return &pageIter{
		storage: s,
		currno:  -1,
		nextno:  pageno,
	}
}

func (iter *pageIter) next() bool {
	if iter.nextno == -1 {
		return false
	} else {
		if iter.nextno >= len(iter.storage.pages) {
			panic(iter.nextno)
		}
		iter.currno = iter.nextno
		iter.nextno = iter.storage.getPage(iter.nextno).Next()
		return true
	}
}

func (iter *pageIter) page() *Page {
	return iter.storage.getPage(iter.currno)
}

func (iter *pageIter) pageno() int {
	return iter.currno
}

// Init initializes the page.
func (p *Page) Init() {
	p.Link(-1)
}

// Add adds an item to this page.
func (p *Page) Add(itemid uint64) {
	itemlen := p.nitems
	p.items[itemlen] = itemid
	p.nitems++
}

// Gets returns a slice of items that are in the page.
func (p *Page) Gets() []uint64 {
	itemlen := p.nitems
	return p.items[:itemlen]
}

// CountItems returns the number of items currently in the page.
func (p *Page) CountItems() int {
	return int(p.nitems)
}

// Next returns the next page number.
func (p *Page) Next() int {
	return int(p.link)
}

// Link remembers next page number.
func (p *Page) Link(next int) {
	p.link = int32(next)
}

// Full returns true if the page is full.
func (p *Page) Full() bool {
	// the first byte is for count and the second for linkage
	return p.nitems == int32(len(p.items))
}

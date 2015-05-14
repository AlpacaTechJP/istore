package lru

import (
	"container/list"
	"sync"

	"github.com/gregjones/httpcache/diskcache"
	"github.com/peterbourgon/diskv"
)

type Cache struct {
	MaxBytes     int
	currentBytes int
	ll           *list.List
	cache        map[string]*list.Element
	mu           sync.RWMutex
	d            *diskcache.Cache
}

type entry struct {
	key   string
	value []byte
}

func New(maxBytes int) *Cache {
	// TODO: not used yet.
	// - Set diskcache when memorycache is set, but asyncronously
	// - async diskcache should be synchronized with the same key
	// - Get tests both in-memory and then diskcache
	// - Delete deletes both, while in-memory set doen't.
	// - Need a way to pass diskcache option, disable diskcache (to pass test)
	d := diskcache.NewWithDiskv(
		diskv.New(diskv.Options{
			BasePath:     "/tmp/istorecache",
			CacheSizeMax: uint64(maxBytes * 10),
		}),
	)
	return &Cache{
		MaxBytes: maxBytes,
		ll:       list.New(),
		cache:    map[string]*list.Element{},
		d:        d,
	}
}

func (c *Cache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ee, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ee)
		c.currentBytes -= len(ee.Value.(*entry).value)
		ee.Value.(*entry).value = value
		c.currentBytes += len(value)
		return
	}
	ele := c.ll.PushFront(&entry{key, value})
	c.currentBytes += len(value)
	c.cache[key] = ele
	for c.MaxBytes != 0 && c.currentBytes > c.MaxBytes {
		ele := c.ll.Back()
		if ele != nil {
			c.removeElement(ele)
		}
	}
}

func (c *Cache) Get(key string) (value []byte, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if ele, hit := c.cache[key]; hit {
		c.ll.MoveToFront(ele)
		return ele.Value.(*entry).value, true
	}
	return
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
	}
}

func (c *Cache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	kv := e.Value.(*entry)
	delete(c.cache, kv.key)
	c.currentBytes -= len(kv.value)
}

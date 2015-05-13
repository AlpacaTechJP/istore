package lru

import (
	"strconv"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type S struct{}

var _ = Suite(&S{})

func (_ *S) TestCache(c *C) {
	cache := New(1024)

	for i := 0; i < 10; i++ {
		val := make([]byte, 100)
		key := strconv.Itoa(i)
		cache.Set(key, val)
	}

	for i := 0; i < 10; i++ {
		key := strconv.Itoa(i)
		val, found := cache.Get(key)
		c.Check(found, Equals, true)
		c.Check(val, Not(Equals), nil)
	}

	cache.Set("big", make([]byte, 900))

	for i := 0; i < 10; i++ {
		key := strconv.Itoa(i)
		_, found := cache.Get(key)
		c.Check(found, Equals, i == 9)
	}
	cache.Delete("9")
	var found bool
	_, found = cache.Get("9")
	c.Check(found, Equals, false)
}

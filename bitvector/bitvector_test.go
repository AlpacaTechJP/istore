package bitvector

import (
	"fmt"
	. "gopkg.in/check.v1"
	"testing"
)

func Test(t *testing.T) { TestingT(t) }

type S struct{}

var _ = Suite(&S{})

func (_ *S) TestBasic(c *C) {
	bv := New(16)
	bv.Set(0)
	bv.Set(3)
	bv.Set(4)
	bv.Set(12)
	c.Check(bv.String(), Equals, "10011000 00001000")

	bv.Unset(12)
	c.Check(bv.String(), Equals, "10011000 00000000")
}

func (_ *S) TestScan(c *C) {
	c.Check(MustScan("01010101 10101010").String(), Equals, "01010101 10101010")
}

func (_ *S) TestUint32(c *C) {
	bv := New(15)
	bv.Set(0)
	c.Check(bv.Uint32(), Equals, uint32(1))

	bv.Set(8)
	c.Check(bv.Uint32(), Equals, uint32(1 << 8 | 1))

	// trim size, bit(14) is ignored.
	bv.Set(14)
	bv2 := FromUint32(bv.Uint32(), 14)
	c.Check(bv2.Uint32(), Equals, uint32(1 << 8 | 1))

	// trim more to lower bytes, bit(8) is also cut off
	bv3 := FromUint32(bv.Uint32(), 8)
	c.Check(bv3.Uint32(), Equals, uint32(1))
}

func (_ *S) TestHamming(c *C) {
	c.Check(
		Hamming(MustScan("11111111"), MustScan("00000000")),
		Equals, 8)
}

func ExampleSortFrom() {
	data := []*BitVector{
		MustScan("00000000"),
		MustScan("11111111"),
		MustScan("11101110"),
		MustScan("00010001"),
	}

	var c *BitVector
	c, _ = Scan("10010001")
	Slice(data).SortFrom(c)
	for _, bv := range data {
		fmt.Println(bv, Hamming(bv, c))
	}

	// Output:
	// 00010001 1
	// 00000000 3
	// 11111111 5
	// 11101110 7
}

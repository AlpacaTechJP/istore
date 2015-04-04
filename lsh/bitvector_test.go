package lsh

import (
	"fmt"
	"testing"
)

func TestString(t *testing.T) {
	bv := NewBitVector(16)
	bv.Set(0)
	bv.Set(3)
	bv.Set(4)
	bv.Set(12)

	var expected, result string
	expected = "10011000 00001000"
	result = bv.String()
	if expected != result {
		t.Fatal("fail ", expected, "!=", result)
	}
}

func TestFromString(t *testing.T) {
	var bv *BitVector

	bv, _ = BitVectorFromString("01010101 10101010")

	var expected, result string
	expected = "01010101 10101010"
	result = bv.String()
	if expected != result {
		t.Fatal("fail", expected, "!=", result)
	}
}

func TestHamming(t *testing.T) {
	var bv1, bv2 *BitVector
	bv1, _ = BitVectorFromString("11111111")
	bv2, _ = BitVectorFromString("00000000")

	var expected, result int
	expected = 8
	result = Hamming(bv1, bv2)
	if expected != result {
		t.Fatal("fail", expected, "!=", result)
	}
}

func ExampleSortFrom() {
	strdata := []string{
		"00000000",
		"11111111",
		"11101110",
		"00010001",
	}

	data := make([]*BitVector, len(strdata))
	for i, str := range strdata {
		data[i], _ = BitVectorFromString(str)
	}

	var c *BitVector
	c, _ = BitVectorFromString("10010001")
	BitVectorSlice(data).SortFrom(c)
	for _, bv := range data {
		fmt.Println(bv, Hamming(bv, c))
	}

	// Output:
	// 00010001 1
	// 00000000 3
	// 11111111 5
	// 11101110 7
}

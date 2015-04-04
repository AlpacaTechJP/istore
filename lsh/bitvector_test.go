package lsh

import (
	"testing"
)

func TestString(t *testing.T) {
	bv := NewBitVector(16)
	bv.Set(0)
	bv.Set(3)
	bv.Set(4)
	bv.Set(12)

	var expected string
	var result string
	expected = "10011000 00001000"
	result = bv.String()
	if expected != result {
		t.Fatal("fail ", expected, "!=", result)
	}
}

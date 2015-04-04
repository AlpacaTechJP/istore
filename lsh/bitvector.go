package lsh

import (
	"bytes"
)

type BitVector struct {
	size int
	bits []uint8
}

func NewBitVector(size int) *BitVector {
	nbytes := size >> 3
	bits := make([]uint8, nbytes, nbytes)
	return &BitVector{
		size: size,
		bits: bits,
	}
}

func (bv *BitVector) Set(i uint) {
	bv.bits[i>>3] |= uint8(0x1) << (i & 0x7)
}

func (bv *BitVector) Get(i uint) bool {
	return (bv.bits[i>>3] & (uint8(0x1) << (i & 0x7))) != 0
}

func (bv *BitVector) String() string {
	buf := new(bytes.Buffer)

	for i := 0; i < bv.size; i++ {
		if i > 0 && i%8 == 0 {
			buf.WriteByte(' ')
		}
		if bv.Get(uint(i)) {
			buf.WriteByte('1')
		} else {
			buf.WriteByte('0')
		}
	}

	return buf.String()
}

func (bv *BitVector) Uint64() uint64 {
	if bv.size <= 8 {
		return uint64(bv.bits[0])
	} else if bv.size <= 16 {
		return uint64((uint64(bv.bits[1]) << 8) | uint64(bv.bits[0]))
	} else if bv.size <= 24 {
		return uint64((uint64(bv.bits[2]) << 16) | (uint64(bv.bits[1]) << 8) | uint64(bv.bits[0]))
	}

	return uint64((uint64(bv.bits[3]) << 24) | (uint64(bv.bits[2]) << 16) | (uint64(bv.bits[1]) << 8) | uint64(bv.bits[0]))
}

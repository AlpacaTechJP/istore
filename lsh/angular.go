package lsh

import (
	"math"

	"github.com/AlpacaDB/istore/bitvector"
)

type Distance interface {
	Distance(x, y []float32) float32
	GetBitVector(vecs [][]float32, v []float32) *bitvector.BitVector
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

func (_ Angular) Side(x, y []float32) bool {
	var dot float32 = 0
	for i := 0; i < len(x); i++ {
		dot += x[i] * y[i]
	}
	return dot > 0
}

func (a Angular) GetBitVector(vecs [][]float32, v []float32) *bitvector.BitVector {
	l := len(vecs)
	bv := bitvector.New(l)

	for i := 0; i < l; i++ {
		if a.Side(vecs[i], v) {
			bv.Set(uint(i))
		}
	}

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

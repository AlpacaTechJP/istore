package lsh

import (
	"math"
	"sort"

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
	// cos_theta = (x dot y) / (norm(x) * norm(y))
	// the denominator does not contribute to the sign, so
	// side = (x dot y) > 0
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

type DistSort struct{
	vecs [][]float32
	cent []float32
	dist Distance
}

func NewDistSort(vecs [][]float32, cent []float32, dist Distance) *DistSort{
	return &DistSort{
		vecs: vecs,
		cent: cent,
		dist: dist,
	}
}

func (s *DistSort) Len() int {
	return len(s.vecs)
}

func (s *DistSort) Less(i, j int) bool {
	dist_i := s.dist.Distance(s.vecs[i], s.cent)
	dist_j := s.dist.Distance(s.vecs[j], s.cent)
	return dist_i < dist_j
}

func (s *DistSort) Swap(i, j int) {
	s.vecs[i], s.vecs[j] = s.vecs[j], s.vecs[i]
}

func (s *DistSort) Sort() {
	sort.Sort(s)
}

// TODO: not implemented.
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

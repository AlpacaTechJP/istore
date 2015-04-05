package lsh

import (
	"fmt"
	"sort"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type S struct{}

var _ = Suite(&S{})

func ExampleSort() {
	data := [][]float32{
		[]float32{1.0, 0.0},
		[]float32{1.0, 0.5},
		[]float32{0.5, 0.5},
		[]float32{-1.0, 0.0},
		[]float32{-1.0, -0.5},
	}

	angular := Angular{}
	cent := []float32{0.3, 0.3}
	sort.Sort(NewDistSort(data, cent, angular))

	for _, v := range data {
		fmt.Printf("(%f,%f) -> %f\n", v[0], v[1], angular.Distance(v, cent))
	}

	// Output:
	// (0.500000,0.500000) -> -0.000000
	// (1.000000,0.500000) -> 0.102633
	// (1.000000,0.000000) -> 0.585786
	// (-1.000000,0.000000) -> 3.414214
	// (-1.000000,-0.500000) -> 3.897367
}

//func (_ *S) TestIndex(c *C) {
//	rng := rand.New(rand.NewSource(0))
//	vecsize := 2
//	data := make([][]float32)
//	for i := 0; i < 1000; i++ {
//		vector := make([]float32, vecsize, vecsize)
//		var sum float64 = 0
//		for j := 0; j < vecsize; j++ {
//			val := rng.NormFloat64()
//			vector[j] = float32(val)
//			sum += val * val
//		}
//		norm := float32(math.Sqrt(sum))
//		for j := 0; j < vecsize; j++ {
//			vector[j] /= norm
//		}
//		data = append(data, vector)
//	}
//}

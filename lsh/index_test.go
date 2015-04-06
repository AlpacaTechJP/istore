package lsh

import (
	"fmt"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type S struct{}

var _ = Suite(&S{})

func displayVecs(vecs [][]float32, cent []float32, dist Distance) {
	for _, v := range vecs {
		fmt.Printf("(%f,%f) -> %f\n", v[0], v[1], dist.Distance(v, cent))
	}
}

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
	NewDistSort(data, cent, angular).Sort()
	displayVecs(data, cent, angular)

	// Output:
	// (0.500000,0.500000) -> -0.000000
	// (1.000000,0.500000) -> 0.102633
	// (1.000000,0.000000) -> 0.585786
	// (-1.000000,0.000000) -> 3.414214
	// (-1.000000,-0.500000) -> 3.897367
}

func ExampleSearch() {
	gen := NewRandomVectorGen(42, 2)
	data := gen.Generate(1000)
	index_data := make([][]float32, len(data), len(data))
	copy(index_data, data)
	cent := []float32{0.3, 0.3}
	angular := Angular{}

	// TODO: bits between 4, 8 and 16 returns the same result??
	index := NewIndexer(39, 8, 2)
	for i, v := range index_data {
		index.Add(uint64(i+1), v)
	}
	items := index.Search(cent, 5)
	fmt.Println("Search: len(items) = ", len(items))

	results := make([][]float32, len(items), len(items))
	for i, itemid := range items {
		results[i] = index_data[itemid-1]
	}

	NewDistSort(data, cent, angular).Sort()
	displayVecs(data[:5], cent, angular)

	fmt.Println("------")
	NewDistSort(results, cent, angular).Sort()
	displayVecs(results[:5], cent, angular)

	// Output:
	// Search: len(items) =  124
	// (0.916952,0.914800) -> 0.000001
	// (1.043253,1.029126) -> 0.000046
	// (0.847707,0.861302) -> 0.000063
	// (0.935449,0.970462) -> 0.000337
	// (0.399547,0.384939) -> 0.000347
	// ------
	// (0.916952,0.914800) -> 0.000001
	// (1.043253,1.029126) -> 0.000046
	// (0.847707,0.861302) -> 0.000063
	// (0.935449,0.970462) -> 0.000337
	// (0.399547,0.384939) -> 0.000347
}

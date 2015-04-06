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
		[]float32{-0.7, -0.7},
	}

	angular := Angular{}
	cent := []float32{0.3, 0.3}
	NewDistSort(data, cent, angular).Sort()
	displayVecs(data, cent, angular)

	// Output:
	// (0.500000,0.500000) -> -0.000000
	// (1.000000,0.500000) -> 0.025658
	// (1.000000,0.000000) -> 0.146447
	// (-1.000000,0.000000) -> 0.853553
	// (-1.000000,-0.500000) -> 0.974342
	// (-0.700000,-0.700000) -> 1.000000
}

func ExampleSearch() {
	// seed = 42
	gen := NewRandomVectorGen(42, 2)
	data := gen.Generate(1000)
	index_data := make([][]float32, len(data), len(data))
	copy(index_data, data)

	// Search from (0.3,0.3)
	cent := []float32{0.3, 0.3}
	angular := Angular{}

	// seed = 39, bitsize = 8
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
	// (0.916952,0.914800) -> 0.000000
	// (1.043253,1.029126) -> 0.000012
	// (0.847707,0.861302) -> 0.000016
	// (0.935449,0.970462) -> 0.000084
	// (0.399547,0.384939) -> 0.000087
	// ------
	// (0.916952,0.914800) -> 0.000000
	// (1.043253,1.029126) -> 0.000012
	// (0.847707,0.861302) -> 0.000016
	// (0.935449,0.970462) -> 0.000084
	// (0.399547,0.384939) -> 0.000087
}

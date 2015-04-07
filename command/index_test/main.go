package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/AlpacaDB/istore/lsh"
)

func readData() [][]float32 {
	decoder := json.NewDecoder(os.Stdin)
	var result [][]float32
	decoder.Decode(&result)
	return result
}

// main dumps index given the set of vectors fed from stdin.
// Use this with randvec like the following
//     randvec -output=json -seed=42 | index_test -seed=0 -bitsize=8
func main() {
	var seed = flag.Int64("seed", 39, "seed to indexer")
	var bitsize = flag.Int("bitsize", 8, "bitsize")
	var limit = flag.Int("limit", 5, "limit")
	flag.Parse()
	data := readData()
	ndim := len(data[0])

	index := lsh.NewIndexer(*seed, *bitsize, ndim)
	for i, v := range data {
		index.Add(uint64(i+1), v)
	}

	fmt.Println(index.Dump())

	if len(flag.Args()) > 0 {
		for _, arg := range flag.Args() {
			vec := make([]float32, ndim, ndim)
			elems := strings.Split(arg, ",")
			for j := 0; j < ndim && j < len(elems); j++ {
				_, err := fmt.Sscanf(elems[j], "%f", &vec[j])
				if err != nil {
					fmt.Println(err)
				}
			}

			bv := index.GetBitVector(vec)
			candidates := index.Candidates(vec, *limit)
			fmt.Println(fmt.Sprintf("Search: %v (bits=%d:%v), len(candidates) = %d", vec, bv.Uint32(), bv, len(candidates)))

			results := index.Qualify(vec, *limit, lsh.SimpleRecords(data), candidates)
			distance := lsh.Angular{}
			for _, v := range results {
				fmt.Printf("%v -> %f\n", v.Vector(), distance.Distance(vec, v.Vector()))
			}
		}
	}
}

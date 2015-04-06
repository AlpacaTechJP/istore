package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

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
	flag.Parse()
	//cent := []float32{0.3, 0.3}
	data := readData()

	index := lsh.NewIndexer(*seed, *bitsize, len(data[0]))
	for i, v := range data {
		index.Add(uint64(i+1), v)
	}

	fmt.Println(index.Dump())
}

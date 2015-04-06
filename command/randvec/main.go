package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/AlpacaDB/istore/lsh"
)

func main() {
	var seed = flag.Int64("seed", 0, "random seed")
	var vdim = flag.Int("vdim", 2, "vector dimension")
	var nvectors = flag.Int("nvectors", 1000, "number of vectors")
	var output = flag.String("output", "csv", "csv|json")

	flag.Parse()

	generator := lsh.NewRandomVectorGen(*seed, *vdim)
	switch *output {
	case "csv":
		generator.GenerateCsv(*nvectors, os.Stdout)
	case "json":
		generator.GenerateJson(*nvectors, os.Stdout)
	default:
		fmt.Println("unexpected output", *output)
		os.Exit(1)
	}
}

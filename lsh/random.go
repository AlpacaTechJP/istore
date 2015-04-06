package lsh

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
)

type RandomVectorGen struct{
	rng *rand.Rand
	vecsize int
}

func NewRandomVectorGen(seed int64, vecsize int) *RandomVectorGen {
	return &RandomVectorGen{
		rng: rand.New(rand.NewSource(seed)),
		vecsize: vecsize,
	}
}

func (r *RandomVectorGen) Next() []float32 {
	vector := make([]float32, r.vecsize, r.vecsize)
	for i := 0; i < r.vecsize; i++ {
		val := r.rng.NormFloat64()
		vector[i] = float32(val)
	}

	return vector
}

func (r *RandomVectorGen) GenerateCsv(n int, w io.Writer) (err error) {
	writer := bufio.NewWriter(w)
	for i := 0; i < n; i++ {
		if i > 0 {
			_, err = writer.WriteString("\n")
		}
		vector := r.Next()
		for j := 0; j < r.vecsize; j++ {
			if j > 0 {
				_, err = writer.WriteString(",")
			}
			_, err = writer.WriteString(fmt.Sprintf("%f", vector[j]))
		}
	}

	return err
}

func (r *RandomVectorGen) GenerateJson(n int, w io.Writer) (err error) {
	list := [][]float32{}
	for i := 0; i < n; i++ {
		vector := r.Next()
		list = append(list, vector)
	}
	encoder := json.NewEncoder(w)
	return encoder.Encode(list)
}

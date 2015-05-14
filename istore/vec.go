package istore

import (
	"bytes"
	"fmt"
	"math"

	"github.com/umitanuki/msgp/msgp"
)

type Vec32 struct {
	Elems []float32
}

func init() {
	msgp.RegisterExtension(
		(&Vec32{}).ExtensionType(),
		func() msgp.Extension { return new(Vec32) })
}

func (v *Vec32) ExtensionType() int8 {
	return 42
}

func (v *Vec32) Len() int {
	return 4 * len(v.Elems)
}

func (v *Vec32) MarshalBinaryTo(b []byte) error {
	for _, e := range v.Elems {
		bits := math.Float32bits(e)
		b[0] = byte(bits >> 24)
		b[1] = byte(bits >> 16)
		b[2] = byte(bits >> 8)
		b[3] = byte(bits)
		b = b[4:]
	}
	return nil
}

func (v *Vec32) UnmarshalBinary(b []byte) error {
	nelems := len(b) / 4
	v.Elems = make([]float32, nelems, nelems)
	for i := 0; i < nelems; i++ {
		bits := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
		v.Elems[i] = math.Float32frombits(bits)
		b = b[4:]
	}
	return nil
}

func (v *Vec32) MarshalJSON() ([]byte, error) {
	buf := new(bytes.Buffer)
	buf.WriteRune('[')
	for i, e := range v.Elems {
		if i > 0 {
			buf.WriteRune(',')
		}
		buf.WriteString(fmt.Sprintf("%f", e))
	}
	buf.WriteRune(']')
	return buf.Bytes(), nil
}

package istore

import (
	"bytes"
	"encoding/json"

	"github.com/tinylib/msgp/msgp"
	. "gopkg.in/check.v1"
)


func (_ *S) TestVec32(c *C) {
	vec := &Vec32{[]float32{0.1, 0.2, 0.3}}
	b := []byte{}
	b, _ = msgp.AppendIntf(b, vec)
	res, _, _ := msgp.ReadIntfBytes(b)

	c.Check(len(res.(*Vec32).Elems), Equals, len(vec.Elems))
	c.Check(res.(*Vec32).Elems, DeepEquals, vec.Elems)

	reader := msgp.NewReader(bytes.NewReader(b))
	jsonbuf := new(bytes.Buffer)
	reader.WriteToJSON(jsonbuf)

	f := []float32{}
	_ = json.Unmarshal(jsonbuf.Bytes(), &f)
	c.Check(f, DeepEquals, vec.Elems)
}

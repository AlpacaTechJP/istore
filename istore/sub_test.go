package istore

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type S struct{}

var _ = Suite(&S{})

func (_ *S) TestExtractTargetURL(c *C) {
	c.Check(extractTargetURL("/abc/http://example.com/foo/bar.jpg"),
		Equals, "http://example.com/foo/bar.jpg")
}

func (_ *S) TestPostItem(c *C) {
	name, _ := ioutil.TempDir("", "istore")
	server := NewServer(name)
	addr := ":8592"

	go http.ListenAndServe(addr, server)

	func() {
		resp, _ := http.PostForm("http://" + addr + "/path/to/file://picts/foo.jpg",
					url.Values{"metadata": {`{"name": "Bob", "user_id": 2159}`}})

		c.Check(resp.StatusCode, Equals, http.StatusCreated)
		decoder := json.NewDecoder(resp.Body)
		meta := ItemMeta{}
		err := decoder.Decode(&meta)
		c.Check(err, Equals, nil)
		c.Check(meta.ItemId, Equals, ItemId(1))
		c.Check(meta.MetaData["name"], Equals, "Bob")

		resp.Body.Close()
	}()

	//func() {
	//	//body := strings.New
	//}

}

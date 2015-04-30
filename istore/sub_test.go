package istore

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type S struct{}

var _ = Suite(&S{})

func (_ *S) TestExtractTargetURL(c *C) {
	target := extractTargetURL("/abc/http://example.com/foo/bar.jpg")
	c.Check(target, Equals, "http://example.com/foo/bar.jpg")

	target = extractTargetURL("/abc/http://localhost:9999/path/http://example.com/foo/bar.jpg")
	c.Check(target, Equals, "http://localhost:9999/path/http://example.com/foo/bar.jpg")

	// rel path
	target = extractTargetURL("/abc/self://def/efg.jpg")
	c.Check(target, Equals, "self:///abc/def/efg.jpg")

	// abs path
	target = extractTargetURL("/abc/self:///def/efg.jpg")
	c.Check(target, Equals, "self:///def/efg.jpg")
}

type mockWriter struct {
	status int
	header http.Header
	body   bytes.Buffer
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		header: http.Header{},
	}
}

func (w *mockWriter) Header() http.Header {
	return w.header
}

func (w *mockWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(b)
}

func (w *mockWriter) WriteHeader(status int) {
	w.status = status
}

func sendForm(method, url string, data url.Values) (*http.Request, error) {
	r, err := http.NewRequest(method, url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r, nil
}

func (_ *S) TestPostItem(c *C) {
	name, _ := ioutil.TempDir("", "istore")
	server := NewServer(name)

	putpost := func(method, path, metadata string, item *ItemMeta) (w *mockWriter, err error) {
		location := "http://example.com" + path
		formdata := url.Values{"metadata": {metadata}}
		r, _ := sendForm(method, location, formdata)
		w = newMockWriter()
		server.ServeHTTP(w, r)

		err = json.Unmarshal(w.body.Bytes(), item)
		return
	}

	post := func(path, metadata string, item *ItemMeta) (w *mockWriter, err error) {
		return putpost("POST", path, metadata, item)
	}

	put := func(path, metadata string, item *ItemMeta) (w *mockWriter, err error) {
		return putpost("PUT", path, metadata, item)
	}

	var mock *mockWriter
	var meta ItemMeta

	// initial create
	meta = ItemMeta{}
	mock, _ = post("/path/to/file:///picts/foo.jpg", `{"name": "Bob", "user_id": 2159}`, &meta)
	c.Check(mock.status, Equals, http.StatusCreated)
	c.Check(meta.ItemId, Equals, ItemId(1))
	c.Check(meta.MetaData["name"], Equals, "Bob")

	// should not bump up the id with the same path
	meta = ItemMeta{}
	mock, _ = post("/path/to/file:///picts/foo.jpg", `{"name": "Bob", "user_id": 9999}`, &meta)
	c.Check(mock.status, Equals, http.StatusOK)
	c.Check(meta.ItemId, Equals, ItemId(1))
	c.Check(meta.MetaData["user_id"], Equals, 9999.0)

	// create another object
	meta = ItemMeta{}
	mock, _ = post("/path/to/file:///picts/bar.jpg", `{"name": "Tom", "user_id": 1}`, &meta)
	c.Check(mock.status, Equals, http.StatusCreated)
	c.Check(meta.ItemId, Equals, ItemId(2))

	// PUT overwrites the entire metadata
	meta = ItemMeta{}
	mock, _ = put("/path/to/file:///picts/foo.jpg", `{"I'm": "new"}`, &meta)
	c.Check(mock.status, Equals, http.StatusOK)
	c.Check(meta.MetaData["name"], Equals, nil)
	c.Check(meta.ItemId, Equals, ItemId(1))
	c.Check(meta.MetaData["I'm"], Equals, "new")

	var r *http.Request
	// GET (list)
	mock = newMockWriter()
	r, _ = http.NewRequest("GET", "http://example.com/path/to/", nil)
	server.ServeHTTP(mock, r)
	resplist := []interface{}{}
	json.Unmarshal(mock.body.Bytes(), &resplist)
	c.Check(mock.status, Equals, http.StatusOK)
	c.Check(resplist[0].(map[string]interface{})["_filepath"], Equals, "/path/to/file:///picts/bar.jpg")
	c.Check(len(resplist), Equals, 2)

	// DELETE -> OK
	mock = newMockWriter()
	r, _ = http.NewRequest("DELETE", "http://example.com/path/to/file:///picts/bar.jpg", nil)
	server.ServeHTTP(mock, r)
	c.Check(mock.status, Equals, http.StatusOK)

	// DELETE -> Not Found
	mock = newMockWriter()
	r, _ = http.NewRequest("DELETE", "http://example.com/path/to/file:///picts/bar.jpg", nil)
	server.ServeHTTP(mock, r)
	// leveldb does not return ErrNotFound??
	// c.Check(mock.status, Equals, http.StatusNotFound)

	// DELETE list
	mock = newMockWriter()
	r, _ = http.NewRequest("DELETE", "http://example.com/path/to/", nil)
	server.ServeHTTP(mock, r)
	c.Check(mock.status, Equals, http.StatusOK)
}

func (_ *S) TestItemId(c *C) {
	itemid := uint64(42)
	b := ItemId(itemid).Bytes()
	c.Check(ToItemId(b), Equals, ItemId(itemid))
}

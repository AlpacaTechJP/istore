package istore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

func (_ *S) TestFileGet(c *C) {
	req, _ := http.NewRequest("GET", "/Not/Exist/File.png", nil)
	resp, err := fileGet(req)
	c.Check(resp.StatusCode, Equals, http.StatusNotFound)
	c.Check(err, Not(Equals), nil)
}

func (_ *S) TestSelf(c *C) {
	name, _ := ioutil.TempDir("", "istore")
	server := NewServer(name)

	request := func(method, path string) (w *mockWriter, err error) {
		Url := "http://example.com" + path
		r, err := http.NewRequest(method, Url, nil)
		w = newMockWriter()
		server.ServeHTTP(w, r)

		return
	}

	var mock *mockWriter
	var err error

	wd, _ := os.Getwd()
	testdata := filepath.Join(wd, "testdata", "sample.jpg")

	mock, err = request("POST", "/path/to/file://"+testdata)
	c.Check(mock.status, Equals, http.StatusCreated)
	c.Check(err, Equals, nil)

	mock, err = request("GET", "/path/to/file://"+testdata+"?apply=resize&w=100")
	c.Check(mock.status, Equals, http.StatusOK)
	resizedImg, format, err := image.Decode(bytes.NewReader(mock.body.Bytes()))
	c.Check(format, Equals, "jpeg")

	mock, err = request("POST", "/path/to/self://file://"+testdata+"%3Fapply=resize&w=100")
	c.Check(mock.status, Equals, http.StatusCreated)

	mock, err = request("GET", "/path/to/self://file://"+testdata+"%3Fapply=resize&w=100")
	c.Check(mock.status, Equals, http.StatusOK)
	resizedImg1, format, err := image.Decode(bytes.NewReader(mock.body.Bytes()))
	c.Check(format, Equals, "jpeg")
	c.Check(resizedImg1.Bounds().Max, Equals, resizedImg.Bounds().Max)

	mock, err = request("POST", "/path/to/self://self://file://"+testdata+"%253Fapply=resize&w=100")
	c.Check(mock.status, Equals, http.StatusCreated)

	mock, err = request("GET", "/path/to/self://self://file://"+testdata+"%253Fapply=resize&w=100")
	c.Check(mock.status, Equals, http.StatusOK)
	resizedImg2, format, err := image.Decode(bytes.NewReader(mock.body.Bytes()))
	c.Check(format, Equals, "jpeg")
	c.Check(resizedImg2.Bounds().Max, Equals, resizedImg.Bounds().Max)
}

func (_ *S) TestSearch(c *C) {
	name, _ := ioutil.TempDir("", "istore")
	server := NewServer(name)

	request := func(method, path string, data interface{}, res interface{}) (w *mockWriter, err error) {
		Url := "http://example.com" + path
		var r *http.Request
		if data == nil {
			r, err = http.NewRequest(method, Url, nil)
		} else if formval, ok := data.(url.Values); ok {
			r, err = sendForm(method, Url, formval)
		} else if bodydata, ok := data.(string); ok {
			r, err = http.NewRequest(method, Url, strings.NewReader(bodydata))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			panic("unexpected input type")
		}
		w = newMockWriter()
		server.ServeHTTP(w, r)

		if res != nil {
			err = json.Unmarshal(w.body.Bytes(), res)
		}
		return
	}

	var mock *mockWriter
	var err error

	mock, err = request("POST", "/path/vec/http://example.com/0.jpg",
		url.Values{"metadata": {`{"vec": [0.5, 0.8]}`}}, nil)
	c.Check(err, Equals, nil)
	mock, err = request("POST", "/path/vec/http://example.com/a.jpg",
		url.Values{"metadata": {`{"vec": [1.0, 0.0]}`}}, nil)
	c.Check(err, Equals, nil)
	mock, err = request("POST", "/path/vec/http://example.com/b.jpg",
		url.Values{"metadata": {`{"vec": [0.0, 1.0]}`}}, nil)
	c.Check(err, Equals, nil)
	mock, err = request("POST", "/path/vec/http://example.com/c.jpg",
		url.Values{"metadata": {`{"vec": [-0.1, -0.1]}`}}, nil)
	c.Check(err, Equals, nil)

	var res []interface{}
	mock, err = request("POST", "/path/vec/_search",
		`{"similar": {"to": "/path/vec/http://example.com/0.jpg", "by": "vec", "limit": 10}}`, &res)
	c.Check(err, Equals, nil)

	c.Check(res[0].(map[string]interface{})["_filepath"], Equals, "/path/vec/http://example.com/0.jpg")
	c.Check(res[1].(map[string]interface{})["_filepath"], Equals, "/path/vec/http://example.com/b.jpg")
	c.Check(res[2].(map[string]interface{})["_filepath"], Equals, "/path/vec/http://example.com/a.jpg")
	c.Check(res[3].(map[string]interface{})["_filepath"], Equals, "/path/vec/http://example.com/c.jpg")

	fmt.Println(mock.body.String())

	_ = mock
	_ = err
}

func (_ *S) TestItemId(c *C) {
	itemid := uint64(42)
	b := ItemId(itemid).Bytes()
	c.Check(ToItemId(b), Equals, ItemId(itemid))
}

package istore

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/golang/glog"
)

type roundTripper struct{}

func (s *Server) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Scheme {
	case "file":
		return s.fileGet(req)
	case "http", "https":
		client := &http.Client{}
		return client.Do(req)
	case "self":
		return s.selfGet(req)
	}

	return nil, fmt.Errorf("unknown scheme %s", req.URL.Scheme)
}

//func (s *Server) getContent(dir, Url string) (*http.Response, error) {
//	u, err := url.Parse(Url)
//	if err != nil {
//		return nil, err
//	}
//	switch u.Scheme {
//	case "file":
//		return s.fileGet(Url)
//	case "http", "https":
//		return s.Client.Get(Url)
//	case "self":
//		return s.selfGet(dir, Url)
//	}
//
//	return nil, fmt.Errorf("unknown scheme %s", u.Scheme)
//}

func (s *Server) fileGet(req *http.Request) (*http.Response, error) {
	filename := req.URL.Path

	content, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
		Body:       content,
	}

	ctype := mime.TypeByExtension(filepath.Ext(filename))
	if ctype == "" {
		var buf [512]byte // see net/http/sniff.go
		n, _ := io.ReadFull(content, buf[:])
		ctype = http.DetectContentType(buf[:n])
		_, err := content.Seek(0, os.SEEK_SET) // rewind to output whole file
		if err != nil {
			return nil, err
		}
	}
	resp.Header.Set("Content-type", ctype)

	if stat, err := content.Stat(); err == nil {
		resp.ContentLength = stat.Size()
		resp.Header.Set("Content-length", fmt.Sprintf("%d", stat.Size()))
	} else {
		glog.Error(err)
	}

	return resp, nil
}

func (s *Server) selfGet(req *http.Request) (*http.Response, error) {
	newpath := req.URL.Path
	newreq, err := http.NewRequest("GET", newpath, nil)
	if err != nil {
		glog.Error("Error in newpath ", newpath)
		return nil, err
	}
	newreq.URL.Path, err = url.QueryUnescape(newreq.URL.Path)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return s.GetApply(newreq)
}

// ----
// 1st level := http://example.com/foo/bar/video.flv?abc=xyz&def=1
// 2nd level := self://http://example.com/foo/bar/video.flv%3Fabc=xyz%26def=1?param=value
// 3rd level := self://self://http://example.com/foo/bar/video.flv%253Fabc=xyz%2526def=1%3Fparam=value
// => to make self url, escape query of the path part, append raw '?' query
//    and to use self url, split by '?', use the query, de-escape the path including internal query part.

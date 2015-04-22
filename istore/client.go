package istore

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
)

func (s *Server) getContent(dir, Url string) (*http.Response, error) {
	u, err := url.Parse(Url)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "file":
		return s.fileGet(Url)
	case "http", "https":
		return s.Client.Get(Url)
	case "self":
		return s.selfGet(dir, Url)
	}

	return nil, fmt.Errorf("unknown scheme %s", u.Scheme)
}

func (s *Server) fileGet(Url string) (*http.Response, error) {
	filename := Url[len("file://"):]

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

func (s *Server) selfGet(dir, Url string) (*http.Response, error) {
	path := Url[len("self://"):]
	if strings.HasPrefix(path, "./") {
		path = path[2:]
	}
	newpath := path
	if !strings.HasPrefix(path, "/") {
		newpath = dir + path
	}
	// TODO: the rule of self scheme seems to be de-escape '%3F' to '?'
	r, err := http.NewRequest("GET", newpath, nil)
	if err != nil {
		glog.Error("Error in newpath ", newpath)
		return nil, err
	}
	return s.GetApply(r)
}

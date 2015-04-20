package istore

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	//"mime"
	"net/http"
	"net/url"
	"os"
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
	filepath := Url[len("file://"):]

	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "HTTP/1.1 200 OK")
	//if idx := strings.LastIndex(filepath, "."); idx > -1 {
	//	ext := filepath[idx:]
	//	fmt.Fprintf(buf, "\nContent-type: %s", mime.TypeByExtension(ext))
	//}
	//if stat, err := f.Stat(); err == nil {
	//	fmt.Fprintf(buf, "\nContent-length: %d", stat.Size())
	//} else {
	//	glog.Error(err)
	//}
	fmt.Fprintf(buf, "\n\n")
	io.Copy(buf, f)

	return http.ReadResponse(bufio.NewReader(buf), nil)
}

func (s *Server) selfGet(dir, Url string) (*http.Response, error) {
	path := Url[len("self://"):]
	if strings.HasPrefix(path, "./") {
		path = path[2:]
	}
	newpath := dir + path
	r, err := http.NewRequest("GET", newpath, nil)
	if err != nil {
		glog.Error("Error in newpath ", newpath)
		return nil, err
	}
	return s.GetApply(r)
}

package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/gregjones/httpcache"
)

type Proxy struct {
	Client *http.Client
	Cache  httpcache.Cache
}

func copyHeader(w http.ResponseWriter, r *http.Response, header string) {
	key := http.CanonicalHeaderKey(header)
	if value, ok := r.Header[key]; ok {
		w.Header()[key] = value
	}
}

func NewProxy() *Proxy {
	cache := httpcache.NewMemoryCache()
	client := &http.Client{}
	client.Transport = httpcache.NewTransport(cache)

	return &Proxy{
		Client: client,
		Cache:  cache,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var urlstr string
	if strings.HasPrefix(r.URL.Path, "/pub/") {
		urlstr = r.URL.Path[5:]
	} else {
		http.NotFound(w, r)
		return
	}
	glog.Info(urlstr)

	client := p.Client
	resp, err := client.Get(urlstr)
	if err != nil {
		var msg string
		statusCode := http.StatusBadRequest
		if resp == nil {
			msg = fmt.Sprintf("%v", err)
		} else {
			msg = fmt.Sprintf("remote URL %q returned status: %v", urlstr, resp.Status)
			statusCode = resp.StatusCode
		}
		glog.Error(msg)
		http.Error(w, msg, statusCode)
		return
	}

	copyHeader(w, resp, "Last-Modified")
	copyHeader(w, resp, "Expires")
	copyHeader(w, resp, "Etag")
	copyHeader(w, resp, "Content-Length")
	copyHeader(w, resp, "Content-Type")
	io.Copy(w, resp.Body)
}

func main() {
	flag.Parse()
	addr := ":8592"
	handler := NewProxy()
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	err := server.ListenAndServe()
	if err != nil {
		glog.Fatal("ListenAndServe: ", err)
	}
}

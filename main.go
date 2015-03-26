package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"

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
	url := r.FormValue("url")
	glog.Info(url)

	client := p.Client
	resp, err := client.Get(url)
	if err != nil {
		msg := fmt.Sprintf("remote URL %q returned status: %v", url, resp.Status)
		glog.Error(msg)
		http.Error(w, msg, resp.StatusCode)
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

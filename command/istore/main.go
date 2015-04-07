package main

import(
	"flag"
	"net/http"

	"github.com/golang/glog"
	"github.com/AlpacaDB/istore/istore"
)

const _DbFile = "/tmp/metadb"

func main() {
	flag.Parse()
	addr := ":8592"
	handler := istore.NewServer(_DbFile)
	glog.Infof("Listening on %v", addr)
	err := http.ListenAndServe(addr, handler)
	if err != nil {
		glog.Fatal("ListenAndServe: ", err)
	}
}

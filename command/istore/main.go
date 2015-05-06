package main

import (
	"flag"
	"net/http"

	"github.com/AlpacaDB/istore/istore"
	"github.com/golang/glog"
)

func main() {
	laddr := flag.String("l", ":8592", "listen address")
	dbfile := flag.String("d", "/tmp/metadb", "datagbase file path")
	flag.Parse()
	handler := istore.NewServer(*dbfile)
	glog.Infof("Listening on %v using DB at %v", *laddr, *dbfile)
	err := http.ListenAndServe(*laddr, handler)
	if err != nil {
		glog.Fatal("ListenAndServe: ", err)
	}
}

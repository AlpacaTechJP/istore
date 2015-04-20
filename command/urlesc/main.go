package main

import (
	"flag"
	"fmt"
	"net/url"
)

func main() {
	flag.Parse()
	for _, u := range flag.Args() {
		fmt.Println(url.QueryEscape(u))
	}
}

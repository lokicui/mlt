package main

import (
	"flag"
	"fmt"
	"github.com/lokicui/mlt/g"
	"github.com/lokicui/mlt/http"
	"os"
)

func main() {
	flag.Parse()
	if *g.Version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}
	addr := *g.Addr + ":" + fmt.Sprintf("%d", *g.Port)
	http.Start(addr)
}

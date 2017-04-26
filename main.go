package main

import (
	"flag"
	"fmt"
	"github.com/lokicui/mlt/g"
    "github.com/lokicui/mlt/http"
	"os"
)

var gAddr = flag.String("addr", "0.0.0.0", "Specify local addr for remote connects")
var gPort = flag.Int("port", 8080, "Specify local port for remote connects")
var gDebug = flag.Int("debug", 0, "debug level, 1 for debug")
var gVersion = flag.Bool("v", false, "show version")

func main() {
	flag.Parse()
	if *gVersion {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}
    addr := *gAddr+":"+fmt.Sprintf("%d", *gPort)
    http.Start(addr)
}

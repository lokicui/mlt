package g

import (
	"flag"
)

const (
	VERSION = "5.1.1"
)

var Addr = flag.String("addr", "0.0.0.0", "Specify local addr for remote connects")
var Port = flag.Int("port", 8080, "Specify local port for remote connects")
var Debug = flag.Int("debug", 0, "debug level, 1 for debug")
var Version = flag.Bool("v", false, "show version")
var ESAddrs = flag.String("esAddrs", "http://10.134.13.99:9200,http://10.134.14.27:9200,http://10.134.28.85:9200", "Specify ES clients for search")

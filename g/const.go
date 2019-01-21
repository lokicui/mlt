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

//var ESAddrs = flag.String("esAddrs", "http://10.134.13.99:9200,http://10.134.14.27:9200,http://10.134.28.85:9200", "Specify ES clients for search")
var ESAddrs = flag.String("esAddrs", "http://rsync.master01.luedonges.sjs.ted:9100,http://rsync.master02.luedonges.sjs.ted:9100,http://rsync.master03.luedonges.sjs.ted:9100", "Specify ES clients for search")
var ESTagInfoAddrs = flag.String("esTagInfoAddrs", "http://rsync.m1.luedonges.sjs.ted:9100,http://rsync.m2.luedonges.sjs.ted:9100,http://rsync.m3.luedonges.sjs.ted:9100", "Specify taginfo ES clients for search")
var ESUser = flag.String("esUser", "admin", "Specify ES user for basic authentication")
var ESPasswd = flag.String("esPasswd", "Admin_ld", "Specify ES user passwd for basic authentication")

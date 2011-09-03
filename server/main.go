package main

import (
	"webrocket"
	"strconv"
	"flag"
	"fmt"
)

const version = "0.0.1"

var (
	action string
	host   string
	port   uint
)

func init() {
	var v bool
	flag.StringVar(&host, "host", "localhost", "serve on given host")
	flag.UintVar(&port, "port", 9772, "listen on given port number")
	flag.BoolVar(&v, "version", false, "display version number")
	flag.Parse()

	if v {
		action = "version"
	}
}

func printVersion() {
	fmt.Printf("Rocket Server v.%s\n", version)
}

func start() {
	addr := host + ":" + strconv.Uitoa(port)
	s := webrocket.NewServer(addr)
	s.Handle("/echo", webrocket.JSONHandler())
	s.ListenAndServe()
}

func main() {
	switch action {
	case "version":
		printVersion()
	default:
		start()
	}
}
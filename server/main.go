package main

import (
	"webrocket"
	"strconv"
	"flag"
	"fmt"
	"log"
	goconf "goconf.googlecode.com/hg"
)

const version = "0.0.1"

type HubConfig struct {
	path        string
	handlerName string
	passphrase  string
}

func (hc *HubConfig) handler() webrocket.Handler {
	switch hc.handlerName {
	case "JSON":
		return webrocket.NewJSONHandler()
	}
	log.Fatalf("Invalid handler type: %s\n", hc.handlerName)
	return nil
}

type ServerConfig struct {
	host string
	port int
	hubs map[string]*HubConfig
}

func (sc *ServerConfig) addr() string {
	return sc.host + ":" + strconv.Itoa(sc.port)
}

var (
	config ServerConfig = ServerConfig{hubs: make(map[string]*HubConfig)}
	action string
	)

func init() {
	var host string
	var port int

	flag.StringVar(&host, "host", "localhost", "serve on given host")
	flag.IntVar(&port, "port", 9772, "listen on given port number")
	flag.Bool("version", false, "display version number")
	flag.Parse()

	flag.Usage = printUsage

	if (flag.NArg() == 1) {
		fname := flag.Arg(0)
		c, err := goconf.ReadConfigFile(fname)
		if err != nil {
			log.Fatalf("Config error: %s\n", err.String())
		}
		for _, s := range c.GetSections() {
			switch s {
			case "default":
				break
			case "server":
				config.host, _ = c.GetString("server", "host")
				config.port, _ = c.GetInt("server", "port")
			default:
				hc := HubConfig{path: s}
				hc.passphrase, _ = c.GetString(s, "passphrase")
				hc.handlerName, _ = c.GetString(s, "handler")
				config.hubs[s] = &hc
			}
		}
		log.Printf("Configured using %s\n", fname)
		action = "serve"
	}

	flag.Visit(func (flag *flag.Flag) {
		switch flag.Name {
		case "version":
			action = "version"
		case "port":
			config.port = port
		case "host":
			config.host = host
		}
	})
}

func printUsage() {
	fmt.Printf("Usage: rocket [flags] configpath...\n")
	fmt.Printf("       rocket [flags]\n")
	flag.PrintDefaults()
}

func printVersion() {
	fmt.Printf("Rocket Server v.%s\n", version)
}

func startServer() {
	s := webrocket.NewServer(config.addr())
	for _, hub := range config.hubs {
		s.Handle(hub.path, hub.handler())
	}
	s.ListenAndServe()
}

func main() {
	switch action {
	case "serve":
		startServer()
	case "version":
		printVersion()
	default:
		flag.Usage()
	}
}
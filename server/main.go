package main

import (
	"webrocket"
	"strconv"
	"flag"
	"fmt"
	"log"
	"json"
	"os"
)

const version = "0.0.1"

type ServerConfig struct {
	Host        string
	Port        int
	TLSCertFile string
	TLSKeyFile  string
	Hubs        []*HubConfig
}

func (sc *ServerConfig) Addr() string {
	return sc.Host + ":" + strconv.Itoa(sc.Port)
}

type HubConfig struct {
	Path   string
	Secret string
	Codec  string
}

func (hc *HubConfig) Handler() webrocket.Handler {
	switch hc.Codec {
	case "JSON":
		h := webrocket.NewJSONHandler()
		h.Secret = hc.Secret
		return h
	}
	log.Fatalf("Invalid handler type: %s\n", hc.Codec)
	return nil
}

var (
	config ServerConfig
	action string
	)

func init() {
	var flags ServerConfig
	flag.Usage = printUsage
	flag.StringVar(&flags.Host, "host", "localhost", "serve on given host")
	flag.IntVar(&flags.Port, "port", 9772, "listen on given port number")
	flag.StringVar(&flags.TLSCertFile, "tls-cert", "", "path to server certificate")
	flag.StringVar(&flags.TLSKeyFile, "tls-key", "", "server's private key")
	flag.Bool("version", false, "display version number")
	flag.Parse()
	configure(flags)
}

func configure(flags ServerConfig) {
	if (flag.NArg() == 1) {
		fname := flag.Arg(0)
		f, err := os.Open(fname)
		if err != nil {
			log.Fatalf("Config reading error: %s\n", err.String())
		}
		dec := json.NewDecoder(f)
		err = dec.Decode(&config)
		if err != nil {
			log.Fatalf("Config parsing error: %s\n", err.String())
		}
		log.Printf("Configured using %s\n", fname)
		action = "serve"
	}
	flag.Visit(func (flag *flag.Flag) {
		switch flag.Name {
		case "version":
			action = "version"
		case "port":
			config.Port = flags.Port
		case "host":
			config.Host = flags.Host
		case "tls-cert":
			config.TLSCertFile = flags.TLSCertFile
		case "tls-key":
			config.TLSKeyFile = flags.TLSKeyFile
		}
	})
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: rocket [flags] configpath...\n")
	fmt.Fprintf(os.Stderr, "       rocket [flags]\n")
	flag.PrintDefaults()
}

func printVersion() {
	fmt.Printf("Rocket Server v.%s\n", version)
}

func startServer() {
	s := webrocket.NewServer(config.Addr())
	for _, hub := range config.Hubs {
		s.Handle(hub.Path, hub.Handler())
	}
	if config.TLSCertFile != "" && config.TLSKeyFile != "" {
		s.ListenAndServeTLS(config.TLSCertFile, config.TLSKeyFile)
	} else {
		s.ListenAndServe()
	}
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
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

const version = "0.2.0"

type ServerConfig struct {
	Host        string
	Port        int
	TLSCertFile string
	TLSKeyFile  string
	Log         string
	Hubs        []*HubConfig
}

func (sc *ServerConfig) Addr() string {
	return sc.Host + ":" + strconv.Itoa(sc.Port)
}

func (sc *ServerConfig) CreateServer() *webrocket.Server {
	s := webrocket.NewServer(sc.Addr())
	if sc.Log != "" {
		logFile := openLogFile(sc.Log)
		s.Log = log.New(logFile, "S : ", log.LstdFlags)
	}
	return s
}

type HubConfig struct {
	Path    string
	Users   map[string]*webrocket.User
	Codec   string
	Log     string
}

func (hc *HubConfig) CreateHandler() webrocket.Handler {
	var logger *log.Logger
	if hc.Log != "" {
		logFile := openLogFile(hc.Log)
		logger = log.New(logFile, hc.Path+" : ", log.LstdFlags)
	}
	switch hc.Codec {
	case "JSON":
		h := webrocket.NewJSONHandler()
		h.Users = hc.Users
		h.Log = logger
		return h
	}
	log.Fatalf("Invalid handler type: %s\n", hc.Codec)
	return nil
}

var (
	server ServerConfig
	action string
)

func init() {
	var flags ServerConfig
	flag.Usage = printUsage
	flag.StringVar(&flags.Host, "host", "localhost", "serve on given host")
	flag.IntVar(&flags.Port, "port", 9772, "listen on given port number")
	flag.StringVar(&flags.TLSCertFile, "tls-cert", "", "path to server certificate")
	flag.StringVar(&flags.TLSKeyFile, "tls-key", "", "server's private key")
	flag.StringVar(&flags.Log, "log", "", "path to log file")
	flag.Bool("version", false, "display version number")
	flag.Parse()
	configure(flags)
}

func configure(flags ServerConfig) {
	if flag.NArg() == 1 {
		fname := flag.Arg(0)
		f, err := os.Open(fname)
		if err != nil {
			log.Fatalf("Config reading error: %s\n", err.String())
		}
		dec := json.NewDecoder(f)
		err = dec.Decode(&server)
		if err != nil {
			log.Fatalf("Config parsing error: %s\n", err.String())
		}
		action = "serve"
	}
	flag.Visit(func(flag *flag.Flag) {
		switch flag.Name {
		case "version":
			action = "version"
		case "port":
			server.Port = flags.Port
		case "host":
			server.Host = flags.Host
		case "tls-cert":
			server.TLSCertFile = flags.TLSCertFile
		case "tls-key":
			server.TLSKeyFile = flags.TLSKeyFile
		case "log":
			server.Log = flags.Log
		}
	})
}

func openLogFile(fname string) *os.File {
	f, err := os.OpenFile(fname, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("Log file error: %s", err.String())
	}
	return f
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
	s := server.CreateServer()
	for _, hub := range server.Hubs {
		s.Handle(hub.Path, hub.CreateHandler())
	}
	if server.TLSCertFile != "" && server.TLSKeyFile != "" {
		s.ListenAndServeTLS(server.TLSCertFile, server.TLSKeyFile)
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

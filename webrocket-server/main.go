// This package implements executable for starting and preconfiguring
// single webrocket server node.
//
// Copyright (C) 2011 by Krzysztof Kowalik <chris@nu7hat.ch>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.
package main

import (
	"../webrocket"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

type Config struct {
	WsHost   string
	WsPort   uint
	BackHost string
	BackPort uint
	CertFile string
	KeyFile  string
}

var (
	conf Config
	ctx  *webrocket.Context
)

func init() {
	flag.StringVar(&conf.WsHost, "websocket-host", "", "bind websocket endpoint with given interface")
	flag.UintVar(&conf.WsPort, "websocket-port", 8080, "websocket endpoint will listen on this port")
	flag.StringVar(&conf.BackHost, "backend-host", "", "bind backend endpoint with given interface")
	flag.UintVar(&conf.BackPort, "backend-port", 8081, "backend endpoint will listen on this port")
	flag.StringVar(&conf.CertFile, "cert", "", "path to server certificate")
	flag.StringVar(&conf.KeyFile, "key", "", "private key")
	flag.Parse()
}

func SetupEndpoint(kind string, e webrocket.Endpoint) {
	go func() {
		var err error
		if conf.CertFile != "" && conf.KeyFile != "" {
			fmt.Printf(">>> %s is about to listen at `%s`... ", kind, e.Addr())
			err = e.ListenAndServeTLS(conf.CertFile, conf.KeyFile)
		} else {
			fmt.Printf(">>> %s is about to listen at `%s`... ", kind, e.Addr())
			err = e.ListenAndServe()
		}
		if err != nil {
			fmt.Printf("\033[31mFAIL\n!!! %s\033[0m\n", err.Error())
			os.Exit(1)
		}
	}()
	for !e.IsRunning() {
		<-time.After(1e2);
	}
	fmt.Printf("\033[32mOK\033[0m\n")
}

func main() {
	// Setting up a context
	fmt.Printf("... Initializing context\n")
	ctx = webrocket.NewContext()

	// Configuring...
	fmt.Printf("... Loading configuration\n")
	vhost, _ := ctx.AddVhost("/test")
	vhost.OpenChannel("test")

	// Setting up a Backend Workers endpoint
	backend := ctx.NewBackendEndpoint(conf.BackHost, conf.BackPort)
	SetupEndpoint("Backend endpoint", backend);

	// Setting up a Websocket Frontend endpoint
	websocket := ctx.NewWebsocketEndpoint(conf.WsHost, conf.WsPort)
	SetupEndpoint("Websocket endpoint", websocket);

	for {
		// Waiting for the interrupt
		sig := <-signal.Incoming
		if sig == os.SIGKILL || sig == os.SIGINT {
			fmt.Printf("\n... \033[33mInterrupted\033[0m\n")
			return
		}
		if sig == os.SIGTSTP {
			
		}
	}
}
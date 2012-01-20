// This package implements executable for starting and preconfiguring
// single webrocket server node.
//
// Copyright (C) 2011 by Krzysztof Kowalik <chris@nu7hat.ch>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.
package main

import (
	stepper "../gostepper"
	"../webrocket"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	Backend    string
	Websocket  string
	Admin      string
	CertFile   string
	KeyFile    string
	StorageDir string
}

var (
	conf Config
	ctx  *webrocket.Context
	s    stepper.Stepper
)

func init() {
	flag.StringVar(&conf.Websocket, "websocket-addr", ":8080", "websocket endpoint address")
	flag.StringVar(&conf.Backend, "backend-addr", ":8081", "backend endpoint address")
	flag.StringVar(&conf.Admin, "admin-addr", ":8082", "admin endpoint address")
	flag.StringVar(&conf.CertFile, "cert", "", "path to server certificate")
	flag.StringVar(&conf.KeyFile, "key", "", "private key")
	flag.StringVar(&conf.StorageDir, "storage-dir", "/var/lib/webrocket", "path to webrocket's internal data-store")
	flag.Parse()

	conf.StorageDir, _ = filepath.Abs(conf.StorageDir)
}

func SetupContext() {
	s.Start("Setting up a context")
	ctx = webrocket.NewContext()
	s.Ok()
	s.Start("Loading configuration")
	if err := ctx.SetStorage(conf.StorageDir); err != nil {
		s.Fail(err.Error(), true)
	}
	s.Ok()
	s.Start("Generating cookie")
	if err := ctx.GenerateCookie(false); err != nil {
		s.Fail(err.Error(), true)
	}
	s.Ok()
}

func SetupEndpoint(kind string, e webrocket.Endpoint) {
	go func() {
		var err error
		s.Start("Starting %s", kind)
		if conf.CertFile != "" && conf.KeyFile != "" {
			err = e.ListenAndServeTLS(conf.CertFile, conf.KeyFile)
		} else {
			err = e.ListenAndServe()
		}
		if err != nil {
			s.Fail(err.Error(), true)
		}
	}()
	for !e.IsAlive() {
		<-time.After(500 * time.Nanosecond)
	}
	s.Ok()
}

func SignalTrap() {
	for sig := range signal.Incoming {
		if usig, ok := sig.(os.UnixSignal); ok {
			switch usig {
			case os.SIGQUIT, os.SIGINT:
				fmt.Printf("\n\033[33mInterrupted\033[0m\n")
				if ctx != nil {
					fmt.Printf("\n")
					s.Start("Cleaning up")
					if err := ctx.Kill(); err != nil {
						s.Fail(err.Error(), true)
					}
					s.Ok()
				}
				os.Exit(0)
			case os.SIGTSTP:
				syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
			case os.SIGHUP:
				// TODO: reload configuration
			}
		}
	}
}

func SetupDaemon() {
	fmt.Printf("\nWebRocket has been launched!\n")
}

// TODO: move it to webrocket...
func GetNodeName() string {
	x := exec.Command("uname", "-n")
	node, err := x.Output()
	if err != nil {
		panic("can't get node name: " + err.Error())
	}
	return strings.TrimSpace(string(node))
}

func DisplayAsciiArt() {
	fmt.Printf("\n")
	fmt.Printf(
		`            /\                                                                     ` + "\n" +
		`      ,    /  \      o               .        ___---___                    .       ` + "\n" +
		`          /    \            .              .--\        --.     .     .         .   ` + "\n" +
		`         /______\                        ./.;_.\     __/~ \.                       ` + "\n" +
		`   .    |        |                      /;  / '-'  __\    . \                      ` + "\n" +
		`        |        |    .        .       / ,--'     / .   .;   \        |            ` + "\n" +
		`        |________|                    | .|       /       __   |      -O-       .   ` + "\n" +
		`        |________|                   |__/    __ |  . ;   \ | . |      |            ` + "\n" +
		`       /|   ||   |\                  |      /  \\_    . ;| \___|                   ` + "\n" +
		`      / |   ||   | \    .    o       |      \  .~\\___,--'     |           .       ` + "\n" +
		`     /  |   ||   |  \                 |     | . ; ~~~~\_    __|                    ` + "\n" +
		`    /___|:::||:::|___\   |             \    \   .  .  ; \  /_/   .                 ` + "\n" +
		`        |::::::::|      -O-        .    \   /         . |  ~/                  .   ` + "\n" +
		`         \::::::/         |    .          ~\ \   .      /  /~          o           ` + "\n" +
		`   o      ||__||       .                   ~--___ ; ___--~                         ` + "\n" +
		`            ||                        .          ---         .              .      ` + "\n" +
		`            ''                                                                     ` + "\n")
	fmt.Printf("WebRocket v%s\n", webrocket.Version())
	fmt.Printf("Copyright (C) 2011-2012 by Krzysztof Kowalik and folks at Cubox.\n")
	fmt.Printf("Released under the AGPL. See http://www.webrocket.io/ for details.\n\n")
}

func DisplaySystemSettings() {
	fmt.Printf("\n")
	fmt.Printf("Node               : %s\n", GetNodeName())
	fmt.Printf("Cookie             : %s\n", ctx.Cookie())
	fmt.Printf("Data store dir     : %s\n", conf.StorageDir)
	fmt.Printf("Backend endpoint   : tcp://%s\n", conf.Backend)
	fmt.Printf("Websocket endpoint : ws://%s\n", conf.Websocket)
	fmt.Printf("Admin endpoint     : http://%s\n", conf.Admin)
}

func main() {
	DisplayAsciiArt()
	SetupContext()
	SetupEndpoint("backend endpoint", ctx.NewBackendEndpoint(conf.Backend))
	SetupEndpoint("websocket endpoint", ctx.NewWebsocketEndpoint(conf.Websocket))
	SetupEndpoint("admin endpoint", ctx.NewAdminEndpoint(conf.Admin))
	DisplaySystemSettings()
	SetupDaemon()
	SignalTrap()
}

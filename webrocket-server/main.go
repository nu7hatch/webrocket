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
	"webrocket"
	"flag"
)

type Config struct {
	WsAddr   string
	CtlAddr  string
	CertFile string
	KeyFile  string
}

var conf Config

func init() {
	flag.StringVar(&conf.WsAddr, "wsaddr", ":9772", "bind server with given address")
	flag.StringVar(&conf.CtlAddr, "ctladdr", "localhost:9773", "bind control interface with given address")
	flag.StringVar(&conf.CertFile, "cert", "", "path to server certificate")
	flag.StringVar(&conf.KeyFile, "key", "", "private key")
	flag.Parse()
}

func main() {
	server := webrocket.NewServer(conf.WsAddr)
	server.BindCtl(conf.CtlAddr)
	if conf.CertFile != "" && conf.KeyFile != "" {
		server.ListenAndServeTLS(conf.CertFile, conf.KeyFile)
	} else {
		server.ListenAndServe()
	}
}
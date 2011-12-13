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
)

type Config struct {
	WsAddr   string
	MqAddr   string
	CtlAddr  string
	CertFile string
	KeyFile  string
}

var conf Config

func init() {
	flag.StringVar(&conf.WsAddr, "wsaddr", ":9772", "bind server with given address")
	flag.StringVar(&conf.MqAddr, "mqaddr", "tcp://*:9773", "bind MQ echange with given address")
	flag.StringVar(&conf.CtlAddr, "ctladdr", "127.0.0.1:9774", "bind control interface with given address")
	flag.StringVar(&conf.CertFile, "cert", "", "path to server certificate")
	flag.StringVar(&conf.KeyFile, "key", "", "private key")
	flag.Parse()
}

func main() {
	ctx := webrocket.NewContext()
	ws := ctx.NewWebsocketServer(conf.WsAddr)
	vhost, _ := ctx.AddVhost("/yoda")
	vhost.AddUser("yoda", "pass", webrocket.PermRead|webrocket.PermWrite)

	mq := ctx.NewMqServer(conf.MqAddr)
	go mq.ListenAndServe()
	
	if conf.CertFile != "" && conf.KeyFile != "" {
		ws.ListenAndServeTLS(conf.CertFile, conf.KeyFile)
	} else {
		ws.ListenAndServe()
	}
}
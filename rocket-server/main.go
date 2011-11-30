// Copyright 2011 Chris Kowalik (nu7hatch). All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.
//
// This package implements executable for starting and configuring
// single webrocket server node.
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
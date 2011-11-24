// Copyright 2011 Chris Kowalik (nu7hatch). All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.
//
// Package webrocket implements advanced WebSocket server with custom
// protocols support. 
package webrocket

import (
	"testing"
	"log"
	"bytes"
)

func NewTestServer() *Server {
	server := NewServer(":9771")
	server.Log = log.New(bytes.NewBuffer([]byte{}), "a", log.LstdFlags)
	return server
}

func TestNewServer(t *testing.T) {
	server := NewTestServer()
	if server.Addr != ":9771" {
		t.Errorf("Expected server addr to be `:9771`, given %s", server.Addr)
	}
}

func TestAddVhost(t *testing.T) {
	server := NewTestServer()
	_, err := server.AddVhost("/echo")
	if err != nil {
		t.Errorf("Expected to add new vhosts, error encountered: %s", err.Error())
	}
}

func TestDeleteVhost(t *testing.T) {
	server := NewTestServer()
	server.AddVhost("/echo")
	server.DeleteVhost("/echo")
	if len(server.Vhosts()) != 0 {
		t.Errorf("Expected to delete vhost")
	}
}

func TestVhosts(t *testing.T) {
	server := NewTestServer()
	server.AddVhost("/foo")
	server.AddVhost("/bar")
	vhosts := server.Vhosts()
	for i, vhost := range []string{"/bar", "/foo"} {
		if vhosts[i] != vhost {
			t.Errorf("Expected to have [/bar /foo] vhosts registered, given %s", vhosts)
		} 
	}
}
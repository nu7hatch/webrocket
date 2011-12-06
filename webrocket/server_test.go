// This package provides a hybrid of MQ and WebSockets server with
// support for horizontal scalability.
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
package webrocket

import (
	"bytes"
	"log"
	"testing"
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
	for i, vhost := range []string{"/foo", "/bar"} {
		if vhosts[i] != vhost {
			t.Errorf("Expected to have [/foo /bar] vhosts registered, given %s", vhosts)
		}
	}
}

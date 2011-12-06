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

func NewTestContext() *Context {
	ctx := NewContext()
	ctx.NewWebsocketServer(":9772")
	ctx.Log = log.New(bytes.NewBuffer([]byte{}), "", log.LstdFlags)
	return ctx
}

func TestAddVhost(t *testing.T) {
	ctx := NewTestContext()
	_, err := ctx.AddVhost("/echo")
	if err != nil {
		t.Errorf("Expected to add new vhosts, error encountered: %s", err.Error())
	}
}

func TestDeleteVhost(t *testing.T) {
	ctx := NewTestContext()
	ctx.AddVhost("/echo")
	ctx.DeleteVhost("/echo")
	if len(ctx.Vhosts()) != 0 {
		t.Errorf("Expected to delete vhost")
	}
}

func TestVhosts(t *testing.T) {
	ctx := NewTestContext()
	ctx.AddVhost("/foo")
	ctx.AddVhost("/bar")
	vhosts := ctx.Vhosts()
	for _, ivhost := range []string{"/foo", "/bar"} {
		ok := false
		for _, jvhost := range vhosts {
			if ivhost == jvhost {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("Expected to have [/foo /bar] vhosts registered, given %s", vhosts)
		}
	}
}

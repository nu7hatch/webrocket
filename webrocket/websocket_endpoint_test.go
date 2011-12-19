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

import "testing"

func TestNewWebsocketEndpoint(t *testing.T) {
	ctx := NewContext()
	e := ctx.NewWebsocketEndpoint("localhost", 9000)
	if e.Addr() != "localhost:9000" {
		t.Errorf("Expected to bing websockets endpoint to localhost:9000")
	}
	if ctx.websocket == nil || ctx.websocket.Addr() != e.Addr() {
		t.Errorf("Expected to register websockets endpoint in the context")
	}
}

func TestWebsocketEndpointRegisterAndUnregisterVhost(t *testing.T) {
	ctx := NewContext()
	e := ctx.NewWebsocketEndpoint("localhost", 9000)
	w := e.(*WebsocketEndpoint)
	v, _ := newVhost(ctx, "/foo")
	w.registerVhost(v)
	h, ok := w.handlers["/foo"]
	if !ok || h == nil || h.vhost.Path() != v.Path() {
		t.Errorf("Expected to register vhost in websocket endpoint")
	}
	w.unregisterVhost(v)
	h, ok = w.handlers["/foo"]
	if ok {
		t.Errorf("Expected to unregister vhost from websocket endpoint")
	}
}

func TestWebsocketRegisterDefinedVhostsOnCreate(t *testing.T) {
	ctx := NewContext()
	v, _ := ctx.AddVhost("/foo")
	e := ctx.NewWebsocketEndpoint("localhost", 9000)
	w := e.(*WebsocketEndpoint)
	h, ok := w.handlers["/foo"]
	if !ok || h == nil || h.vhost.Path() != v.Path() {
		t.Errorf("Expected to register defined vhost when websocket endpoint creates")
	}
}
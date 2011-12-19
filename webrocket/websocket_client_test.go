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

func newTestWebsocketClient() *WebsocketClient {
	ctx := NewContext()
	v, _ := newVhost(ctx, "/foo")
	c := newWebsocketClient(v, nil)
	return c
}

func TestNewWebsocketClient(t *testing.T) {
	c := newTestWebsocketClient()
	if len(c.Id()) != 36 {
		t.Errorf("Expected id to be generated")
	}
	if c.IsAuthenticated() {
		t.Errorf("Expected client to be not authenticated")
	}
	if !c.IsAlive() {
		t.Errorf("Expected new client to be alive")
	}
}

func TestWebsocketClientClearSubscribers(t *testing.T) {
	c := newTestWebsocketClient()
	foo, _ := newChannel("foo")
	foo.addSubscriber(c)
	c.clearSubscriptions()
	if len(c.subscriptions) != 0 {
		t.Errorf("Expected to clear all subscriptions")
	}
}

func TestWebsocketClientAuthenticate(t *testing.T) {
	c := newTestWebsocketClient()
	p := NewPermission(".*")
	c.authenticate(p)
	if !c.IsAuthenticated() {
		t.Errorf("Expected to authenticate the client")
	}
	if c.permission.Token() != p.Token() {
		t.Errorf("Expected to authenticate the client with given permission")
	}
}

func TestWebsocketClientIsAllowed(t *testing.T) {
	c := newTestWebsocketClient()
	p := NewPermission("foo")
	c.authenticate(p)
	if !c.IsAllowed("foo") || c.IsAllowed("bar") {
		t.Errorf("Expected client to properly assert permissions")
	}
}
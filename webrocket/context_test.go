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

func TestNewContext(t *testing.T) {
	ctx := NewContext()
	if ctx.Log() == nil {
		t.Errorf("Expected context logger to be initialized")
	}
}

func TestContextSetLog(t *testing.T) {
	ctx := NewContext()
	ctx.SetLog(nil)
	if ctx.Log() != nil {
		t.Errorf("Expected to set other logger")
	}
}

func TestContextAddVhost(t *testing.T) {
	ctx := NewContext()
	v, err := ctx.AddVhost("/foo")
	if err != nil || v == nil {
		t.Errorf("Expected to add vhost")
	}
	_, err = ctx.AddVhost("/foo")
	if err == nil || err.Error() != "vhost already exists" {
		t.Errorf("Expected error while adding duplicated vhost")
	}
	_, ok := ctx.vhosts["/foo"]
	if !ok {
		t.Errorf("Expected to add vhost")
	}
}

func TestContextAddVhostWhenWebsocketEndpointPresent(t *testing.T) {
	ctx := NewContext()
	e := ctx.NewWebsocketEndpoint("localhost:3000")
	w := e.(*WebsocketEndpoint)
	v, _ := ctx.AddVhost("/foo")
	h, ok := w.handlers["/foo"]
	if !ok || h == nil || h.vhost.Path() != v.Path() {
		t.Errorf("Expected to register vhost in websocket endpoint")
	}
}

func TestContextDeleteVhost(t *testing.T) {
	ctx := NewContext()
	ctx.AddVhost("/foo")
	err := ctx.DeleteVhost("/foo")
	if err != nil {
		t.Errorf("Expected to delete vhost without errors")
	}
	_, ok := ctx.vhosts["/foo"]
	if ok {
		t.Errorf("Expected to delete vhost")
	}
	err = ctx.DeleteVhost("/foo")
	if err == nil || err.Error() != "vhost doesn't exist" {
		t.Errorf("Expected an error while deleting non existent vhost")
	}
}

func TestContextDeleteVhostWhenWebsocketEndpointPresent(t *testing.T) {
	ctx := NewContext()
	e := ctx.NewWebsocketEndpoint("localhost:3000")
	w := e.(*WebsocketEndpoint)
	ctx.AddVhost("/foo")
	ctx.DeleteVhost("/foo")
	_, ok := w.handlers["/foo"]
	if ok {
		t.Errorf("Expected to unregister vhost from websocket endpoint")
	}
}

func TestContextGetVhost(t *testing.T) {
	ctx := NewContext()
	ctx.AddVhost("/foo")
	v, err := ctx.Vhost("/foo")
	if err != nil || v == nil || v.Path() != "/foo" {
		t.Errorf("Expected to get vhost")
	}
	_, err = ctx.Vhost("/bar")
	if err == nil || err.Error() != "vhost doesn't exist" {
		t.Errorf("Expected an error getting non existent vhost")
	}
}

func TestContextVhostsList(t *testing.T) {
	ctx := NewContext()
	ctx.AddVhost("/foo")
	if len(ctx.Vhosts()) != 1 {
		t.Errorf("Expected vhosts list to contain one element")
	}
}

func TestContextCookiesGeneration(t *testing.T) {
	// TODO: ...
}

func TestContextClose(t *testing.T) {
	ctx := NewContext()
	ctx.NewWebsocketEndpoint(":9772")
	if ctx.websocket == nil {
		t.Errorf("Expected to set websocket endpoint")
	}
	ctx.NewBackendEndpoint(":9773")
	if ctx.backend == nil {
		t.Errorf("Expected to set backend endpoint")
	}
	ctx.Close()
	if ctx.backend.IsAlive() || ctx.websocket.IsAlive() {
		t.Errorf("Expected to close and kill all endpoints")
	}
}
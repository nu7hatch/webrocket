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

func TestWebsocketServeMuxCleanPath(t *testing.T) {
	if cleanPath("//") != "/" {
		t.Error("Expected to clean multiple slashes from the path")
	}
	if cleanPath("/foo/../bar/./") != "/bar/" {
		t.Error("Expected to clean dots from the path")
	}
}

func TestWebsocketServeMuxAddHandler(t *testing.T) {
	mux := NewWebsocketServeMux()
	mux.AddHandler("/foo", &websocketHandler{})
	if _, ok := mux.m["/foo"]; !ok {
		t.Errorf("Expected to add handler")
	}
}

func TestWebsocketServeMuxDeleteHandler(t *testing.T) {
	mux := NewWebsocketServeMux()
	mux.AddHandler("/foo", &websocketHandler{})
	if ok := mux.DeleteHandler("/foo"); !ok {
		t.Errorf("Expected to delete handler")
	}
	if _, ok := mux.m["/foo"]; ok {
		t.Errorf("Expected to delete handler")
	}
	if ok := mux.DeleteHandler("/bar"); ok {
		t.Errorf("Expected to not delete non existing handler")
	}
}

func TestWebsocketServeMuxMatch(t *testing.T) {
	mux := NewWebsocketServeMux()
	mux.AddHandler("/foo", &websocketHandler{})
	if handler := mux.Match("/foo"); handler == nil {
		t.Errorf("Expected to match existing handler")
	}
	if handler := mux.Match("/bar"); handler != nil {
		t.Errorf("Expected to not match non existing handler")
	}
}

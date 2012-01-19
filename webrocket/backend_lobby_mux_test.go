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

func TestBackendLobbyMuxAddLobby(t *testing.T) {
	mux := NewBackendLobbyMux()
	mux.AddLobby("/foo", &backendLobby{})
	if _, ok := mux.m["/foo"]; !ok {
		t.Errorf("Expected to add lobby")
	}
}

func TestBackendLobbyMuxDeleteLobby(t *testing.T) {
	mux := NewBackendLobbyMux()
	l := &backendLobby{queue: make(chan interface{})}
	mux.AddLobby("/foo", l)
	if ok := mux.DeleteLobby("/foo"); !ok {
		t.Errorf("Expected to delete lobby")
	}
	if l.IsAlive() {
		t.Errorf("Lobby should be killed when deleting from the mux")
	}
	if _, ok := mux.m["/foo"]; ok {
		t.Errorf("Expected to delete lobby")
	}
	if ok := mux.DeleteLobby("/bar"); ok {
		t.Errorf("Expected to not delete non existing lobby")
	}
}

func TestBackendLobbyMuxMatch(t *testing.T) {
	mux := NewBackendLobbyMux()
	mux.AddLobby("/foo", &backendLobby{})
	if lobby := mux.Match("/foo"); lobby == nil {
		t.Errorf("Expected to match existing lobby")
	}
	if lobby := mux.Match("/bar"); lobby != nil {
		t.Errorf("Expected to not match non existing lobby")
	}
}

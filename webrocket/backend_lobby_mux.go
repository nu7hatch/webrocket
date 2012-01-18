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

import "sync"

// BackendLobbyMux is an request multiplexer for backend lobbys.
type BackendLobbyMux struct {
	// Handler (lobby) map.
	m map[string]*backendLobby
	// Internal semaphore.
	mtx sync.Mutex
}

// Exported constructors
// -----------------------------------------------------------------------------

// NewBackendLobbyMux allocates and returns a new BackendLobbyMux.
func NewBackendLobbyMux() *BackendLobbyMux {
	return &BackendLobbyMux{m: make(map[string]*backendLobby)}
}

// Exported
// -----------------------------------------------------------------------------

// Match finds a lobby on a handler map given a path string.
//
// p - A path to match against.
//
// Returns matching lobby.
func (mux *BackendLobbyMux) Match(path string) *backendLobby {
	mux.mtx.Lock()
	defer mux.mtx.Unlock()
	h, _ := mux.m[path]
	return h
}

// AddLobby registers the lobby for the given pattern.
//
// pattern - A path to which this lobby will be bound
// lobby - A http lobby that will be registered
//
// Examples:
//
//    helloLobby = websocket.Lobby(func(ws *websocket.Conn) { /* ... */ })
//    mux.AddLobby("/hello", helloLobby)
//
func (mux *BackendLobbyMux) AddLobby(pattern string, lobby *backendLobby) {
	if pattern != "" {
		mux.mtx.Lock()
		defer mux.mtx.Unlock()
		mux.m[pattern] = lobby
	}
}

// DeleteLobby removes matching lobby from the lobby map.
//
// pattern - A pattern under which the lobby has been registered.
//
// Examples:
//
//    mux.DeleteLobby("/hello")
//
// Returns whether the lobby has been removed or not.
func (mux *BackendLobbyMux) DeleteLobby(pattern string) bool {
	if pattern != "" {
		mux.mtx.Lock()
		defer mux.mtx.Unlock()
		if l, ok := mux.m[pattern]; ok {
			l.Kill()
			delete(mux.m, pattern)
			return true
		}
	}
	return false
}

// KillAll terminates all registered lobbys..
func (mux *BackendLobbyMux) KillAll() {
	mux.mtx.Lock()
	defer mux.mtx.Unlock()
	for _, l := range mux.m {
		l.Kill()
	}
}

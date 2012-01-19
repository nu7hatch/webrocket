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
	"net/http"
	"path"
	"sync"
)

// WebsocketServeMux is an HTTP request multiplexer. Basically works the same
// as the http.WebsocketServeMux from the standar library, but allows for
// dynamic adding and removing handlers.
type WebsocketServeMux struct {
	// Handler map.
	m map[string]*websocketHandler
	// Internal semaphore.
	mtx sync.Mutex
}

// Exported constructors
// -----------------------------------------------------------------------------

// NewWebsocketServeMux allocates and returns a new WebsocketServeMux.
func NewWebsocketServeMux() *WebsocketServeMux {
	return &WebsocketServeMux{m: make(map[string]*websocketHandler)}
}

// Internal
// -----------------------------------------------------------------------------

// cleanPath Return the canonical path for p, eliminating `.` and `..` elements.
//
// p - A path to clean
//
// Returns cleaned path.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

// Exported
// -----------------------------------------------------------------------------

// ServeHTTP dispatches the request to the handler whose pattern
// matches the request URL.
//
// w - The response writer
// r - The request to handle
//
func (mux *WebsocketServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean path to canonical form and redirect.
	if p := cleanPath(r.URL.Path); p != r.URL.Path {
		w.Header().Set("Location", p)
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}
	// Host-specific pattern takes precedence over generic ones
	h := mux.Match(r.Host + r.URL.Path)
	if h == nil {
		h = mux.Match(r.URL.Path)
	}
	if h == nil {
		http.NotFoundHandler().ServeHTTP(w, r)
	} else {
		h.ServeHTTP(w, r)
	}
}

// Match finds a handler on a handler map given a path string.
//
// p - A path to match against
//
// Returns matching handler.
func (mux *WebsocketServeMux) Match(path string) *websocketHandler {
	mux.mtx.Lock()
	defer mux.mtx.Unlock()
	h, _ := mux.m[path]
	return h
}

// AddHandler registers the handler for the given pattern.
//
// pattern - A path to which this handler will be bound
// handler - A http handler that will be registered
//
// Examples:
//
//    helloHandler = websocket.Handler(func(ws *websocket.Conn) { /* ... */ })
//    mux.AddHandler("/hello", helloHandler)
//
func (mux *WebsocketServeMux) AddHandler(pattern string, handler *websocketHandler) {
	if pattern != "" {
		mux.mtx.Lock()
		defer mux.mtx.Unlock()
		mux.m[pattern] = handler
	}
}

// DeleteHandler removes matching handler from the handler map.
//
// pattern - A pattern under which the handler has been registered.
//
// Examples:
//
//    mux.DeleteHandler("/hello")
//
// Returns whether the handler has been removed or not.
func (mux *WebsocketServeMux) DeleteHandler(pattern string) bool {
	if pattern != "" {
		mux.mtx.Lock()
		defer mux.mtx.Unlock()
		if h, ok := mux.m[pattern]; ok {
			h.Kill()
			delete(mux.m, pattern)
			return true
		}
	}
	return false
}

// KillAll terminates all registered handlers.
func (mux *WebsocketServeMux) KillAll() {
	mux.mtx.Lock()
	defer mux.mtx.Unlock()
	for _, h := range mux.m {
		h.Kill()
	}
}

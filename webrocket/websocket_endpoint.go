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
	"fmt"
	"net"
	"net/http"
	"path"
	"sync"
)

// WebsocketEndpoint defines parameters for running an WebSocket server.
type WebsocketEndpoint struct {
	http.Server
	BaseEndpoint
	handlers map[string]*websocketHandler
}

// Creates new websockets endpoint bound to specified host and port.
// Leave host blank if you want to bind to all interfaces.
//
// A trivial example server:
// 
//     package main
//      
//     import "webrocket"
//     
//     func main() {
//         ctx := webrocket.NewContext()
//         ws := ctx.NewWebsocketEndpoint("localhost", 8080)
//         // ... configure vhosts and users
//         ws.ListenAndServe()
//     }
//
func (ctx *Context) NewWebsocketEndpoint(host string, port uint) Endpoint {
	e := &WebsocketEndpoint{}
	e.ctx = ctx
	e.handlers = make(map[string]*websocketHandler)
	e.Server.Addr = fmt.Sprintf("%s:%d", host, port)
	e.Handler = NewServeMux()
	ctx.websocket = e
	for _, vhost := range ctx.vhosts {
		e.registerVhost(vhost)
	}
	return e
}

// Registers a websockets handler for specified vhost. 
func (w *WebsocketEndpoint) registerVhost(vhost *Vhost) {
	h := newWebsocketHandler(vhost)
	w.Handler.(*ServeMux).AddHandler(vhost.Path(), h)
	w.handlers[vhost.Path()] = h
	h.start()
}

// Removes websockets handler for specified vhost. 
func (w *WebsocketEndpoint) unregisterVhost(vhost *Vhost) {
	h, ok := w.handlers[vhost.Path()]
	if !ok {
		return
	}
	h.stop()
	delete(w.handlers, vhost.Path())
	w.Handler.(*ServeMux).DeleteHandler(vhost.Path())
}

// Returns address to which this endpoint is bound.
func (w *WebsocketEndpoint) Addr() string {
	return w.Server.Addr
}

// Extendended http.Server.ListenAndServe funcion.
func (w *WebsocketEndpoint) ListenAndServe() error {
	addr := w.Server.Addr
	if addr == "" {
		addr = ":http"
	}
	l, e := net.Listen("tcp", addr)
	if e != nil {
		return e
	}
	w.alive()
	return w.Server.Serve(l)
}

// Extendended http.Server.ListenAndServeTLS funcion.
func (w *WebsocketEndpoint) ListenAndServeTLS(certFile, certKey string) error {
	w.alive()
	return w.Server.ListenAndServeTLS(certFile, certKey)
}

// Extended version of the kill func. Stops all alive handlers.
func (w *WebsocketEndpoint) kill() {
	for _, h := range w.handlers {
		h.stop()
	}
	w.BaseEndpoint.kill()
}

// ServeMux is an HTTP request multiplexer. Basically works the same as
// the http.ServeMux from the standar library, but allows for dynamic adding
// and removing handlers.
type ServeMux struct {
	m   map[string]http.Handler
	mtx sync.Mutex
}

// NewServeMux allocates and returns a new ServeMux.
func NewServeMux() *ServeMux {
	return &ServeMux{m: make(map[string]http.Handler)}
}

// Does path match pattern?
func pathMatch(pattern, path string) bool {
	if len(pattern) == 0 {
		// should not happen
		return false
	}
	n := len(pattern)
	if pattern[n-1] != '/' {
		return pattern == path
	}
	return len(path) >= n && path[0:n] == pattern
}

// Return the canonical path for p, eliminating . and .. elements.
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

// Find a handler on a handler map given a path string
// Most-specific (longest) pattern wins
func (mux *ServeMux) match(path string) http.Handler {
	var h http.Handler
	var n = 0
	for k, v := range mux.m {
		if !pathMatch(k, path) {
			continue
		}
		if h == nil || len(k) > n {
			n = len(k)
			h = v
		}
	}
	return h
}

// ServeHTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean path to canonical form and redirect.
	if p := cleanPath(r.URL.Path); p != r.URL.Path {
		w.Header().Set("Location", p)
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}
	// Host-specific pattern takes precedence over generic ones
	h := mux.match(r.Host + r.URL.Path)
	if h == nil {
		h = mux.match(r.URL.Path)
	}
	if h == nil {
		h = http.NotFoundHandler()
	}
	h.ServeHTTP(w, r)
}

// AddHandler registers the handler for the given pattern.
func (mux *ServeMux) AddHandler(pattern string, handler http.Handler) {
	if pattern == "" {
		panic("http: invalid pattern " + pattern)
	}
	mux.mtx.Lock()
	defer mux.mtx.Unlock()
	mux.m[pattern] = handler
	// Helpful behavior:
	// If pattern is /tree/, insert permanent redirect for /tree.
	n := len(pattern)
	if n > 0 && pattern[n-1] == '/' {
		mux.m[pattern[0:n-1]] = http.RedirectHandler(pattern, http.StatusMovedPermanently)
	}
}

// DeleteHandler closes all connections processed by matching
// handler  and removes is from the server.
func (mux *ServeMux) DeleteHandler(pattern string) bool {
	if pattern == "" {
		panic("http: invalid pattern " + pattern)
	}
	mux.mtx.Lock()
	defer mux.mtx.Unlock()
	_, ok := mux.m[pattern]
	if !ok {
		return false
	}
	delete(mux.m, pattern)
	return true
}

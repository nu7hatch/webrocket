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
	"log"
	"net/http"
	"path"
	"websocket"
)

// Wrapper for standard websocket.Conn structure. Provides additional
// information about connection and maintains sessions. 
type wsConn struct {
	*conn
	*websocket.Conn
	token    string
	channels map[*Channel]bool
}

// wrapWsConn wraps standard websocket connection object into one
// adjusted for webrocket server funcionalities.
func wrapWsConn(ws *websocket.Conn, vhost *Vhost) *wsConn {
	c := &wsConn{Conn: ws, token: generateUniqueToken()}
	c.conn = &conn{vhost: vhost}
	c.channels = make(map[*Channel]bool)
	return c
}

// A helper for quick sending encoded payloads to the connected client.
func (c *wsConn) send(data interface{}) error {
	err := c.vhost.codec.Send(c.Conn, data)
	if err != nil {
		c.vhost.Log.Printf("ws[%s]: ERR_NOT_SEND %s", c.vhost.path, err.Error())
	}
	return err
}

// Unsubscribes this client from all channels.
func (c *wsConn) unsubscribeAll() {
	for ch := range c.channels {
		ch.subscribe <- subscription{c, false}
	}
}

// WebsocketServer defines parameters for running an WebSocket server.
type WebsocketServer struct {
	http.Server
	Log      *log.Logger
	ctx      *Context
	certFile string
	keyFile  string
}

// Creates new websockets server bound to specified addr.
// A Trivial example server:
// 
//     package main
//      
//     import "webrocket"
//     
//     func main() {
//         ctx := webrocket.NewContext()
//         srv := ctx.NewWebsocketServer("localhost:8080")
//         ctx.AddVhost("/echo")
//         srv.ListenAndServe()
//     }
//
func (ctx *Context) NewWebsocketServer(addr string) *WebsocketServer {
	s := &WebsocketServer{ctx: ctx}
	s.Addr, s.Handler = addr, NewServeMux() 
	s.Log = ctx.Log
	ctx.wsServ = s
	return s
}

// Listens on the TCP network address srv.Addr and handles requests on incoming
// websocket connections.
func (s *WebsocketServer) ListenAndServe() error {
	s.Log.Printf("server[ws]: About to listen on %s\n", s.Addr)
	err := s.Server.ListenAndServe()
	if err != nil {
		s.Log.Fatalf("server[ws]: Startup error: %s\n", err.Error())
	}
	return err
}

// Listens on the TCP network address srv.Addr and handles requests on incoming TLS
// websocket connections.
func (s *WebsocketServer) ListenAndServeTLS(certFile, keyFile string) error {
	s.Log.Printf("server[ws]: About to listen on %s", s.Addr)
	err := s.Server.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		s.Log.Fatalf("server[ws]: Secured server startup error: %s\n", err.Error())
	}
	return err
}

// ServeMux is an HTTP request multiplexer. Basically works the same as
// the http.ServeMux from the standar library, but allows for dynamic adding
// and removing handlers.
type ServeMux struct {
	m map[string]http.Handler
}

// NewServeMux allocates and returns a new ServeMux.
func NewServeMux() *ServeMux {
	return &ServeMux{make(map[string]http.Handler)}
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
	_, ok := mux.m[pattern]
	if !ok {
		return false
	}
	delete(mux.m, pattern)
	return true
}

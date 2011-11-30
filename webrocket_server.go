// Copyright 2011 Chris Kowalik (nu7hatch). All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.
//
// Package webrocket implements advanced WebSocket server with custom
// protocols support. 
package webrocket

import (
	"net/http"
	"log"
	"os"
	"fmt"
	"path"
	"errors"
)

// Server defines parameters for running an WebSocket server.
type Server struct {
	http.Server
	Log      *log.Logger
	vhosts   map[string]*Vhost
	certFile string
	keyFile  string
	ctl      string
}

// Creates new rocket's server bound to specified addr.
// A Trivial example server:
// 
//     package main
//      
//     import "webrocket"
//     
//     func main() {
//         s := webrocket.NewServer("ws://localhost:8080")
//         s.AddVhost("/echo")
//         s.ListenAndServe()
//     }
//
func NewServer(addr string) *Server {
	s := new(Server)
	s.Addr, s.Handler = addr, NewServeMux()
	s.vhosts = make(map[string]*Vhost)
	s.Log = log.New(os.Stderr, "", log.LstdFlags)
	return s
}

// Registers new vhost within the server instance.
// 
//     s := webrocket.NewServer(":8080")
//     s.AddVhost("/hello")
//     s.AddVhost("/world")
//     s.ListenAndServe()
//
func (s *Server) AddVhost(path string) (*Vhost, error) {
	vhost := NewVhost(path)
	vhost.Log = s.Log
	s.vhosts[path] = vhost		
	s.Handler.(*ServeMux).AddHandler(path, vhost)
	s.Log.Printf("server: ADD_VHOST path='%s'", path)
	return vhost, nil
}

// Deletes specified vhost from the serve mux.
func (s *Server) DeleteVhost(path string) error {
	vhost, ok := s.vhosts[path]
	if !ok {
		return errors.New(fmt.Sprintf("Vhost `%s` doesn't exist.", path))
	}
	s.Handler.(*ServeMux).DeleteHandler(path)
	vhost.Stop()
	delete(s.vhosts, path)
	s.Log.Printf("server: DELETE_VHOST path='%s'", path)
	return nil
}

// Returns array with registered vhost paths.
func (s *Server) Vhosts() (vhosts []string) {
	vhosts, i := make([]string, len(s.vhosts)), 0
	for path, _ := range s.vhosts {
		vhosts[i] = path
		i += 1
	}
	return vhosts
}

// Binds control interface with specified address.
func (s *Server) BindCtl(addr string) {
	s.ctl = newCtl(addr)
}

// Listens on the TCP network address srv.Addr and handles requests on incoming
// websocket connections.
func (s *Server) ListenAndServe() error {
	s.Log.Printf("server: About to listen on %s\n", s.Addr)
	err := s.Server.ListenAndServe()
	if err != nil {
		s.Log.Fatalf("server: Startup error: %s\n", err.Error())
	}
	return err
}

// Listens on the TCP network address srv.Addr and handles requests on incoming TLS
// websocket connections.
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	s.Log.Printf("server: About to listen on %s", s.Addr)
	err := s.Server.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		s.Log.Fatalf("server: Secured server startup error: %s\n", err.Error())
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
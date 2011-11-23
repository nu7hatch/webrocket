// Copyright 2011 Chris Kowalik (nu7hatch). All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.
//
// Package webrocket implements advanced WebSocket server with custom
// protocols support. 
package webrocket

import (
	"websocket"
	"http"
	"log"
	"os"
)

// handlerMap is a map of resource handlers.
type handlerMap map[string]Handler

// Server defines parameters for running an WebSocket server.
type Server struct {
	http.Server
	Log      *log.Logger
	handlers handlerMap
	certFile string
	keyFile  string
}

/*
Creates new rocket's server bound to specified addr.
A Trivial example server:

    package main

    import "rocket"

    func main() {
         s := rocket.NewServer("ws://localhost:8080")
         s.Handle("/echo", rocket.NewJSONHandler())
         s.ListenAndServe()
    }
*/
func NewServer(addr string) *Server {
	s := new(Server)
	s.Addr, s.Handler = addr, http.NewServeMux()
	s.handlers = make(handlerMap)
	s.Log = log.New(os.Stderr, "S : ", log.LstdFlags)
	return s
}

/*
Registers payload handler under specified path. Handler have to implement
communication protocol callbacks.
*/
func (s *Server) Handle(path string, h Handler) {
	proxy, ok := h.Register(s, path)
	if ok == nil {
		s.Handler.(*http.ServeMux).Handle(path, proxy)
		s.handlers[path] = h
	}
}

/*
Listens on the TCP network address srv.Addr and handles requests on incoming
websocket connections.
*/
func (s *Server) ListenAndServe() os.Error {
	s.Log.Printf("About to listen on %s\n", s.Addr)
	err := s.Server.ListenAndServe()
	if err != nil {
		s.Log.Fatalf("Server startup error: %s\n", err.String())
	}
	return err
}

/*
Listens on the TCP network address srv.Addr and handles requests on incoming TLS
websocket connections.
*/
func (s *Server) ListenAndServeTLS(certFile, keyFile string) os.Error {
	s.Log.Printf("About to listen on %s", s.Addr)
	err := s.Server.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		s.Log.Fatalf("Secured server startup error: %s\n", err.String())
	}
	return err
}

// subscriberMap contains sockets subscribing given channel.
type subscriberMap map[*conn]int

/*
channel keeps information about specified channel and it's subscriptions.
It's hub is used to broadcast messages.
*/
type channel struct {
	name        string
	owner       *handler
	subscribers subscriberMap
	subscribe   chan subscription
	broadcast   chan broadcaster
}

// channelMap is a map of channels.
type channelMap map[string]*channel

// broadcaster is a function for spreading messages to all chanel's subscribers. 
type broadcaster func(ws *conn)

/*
subscription struct is used to modify channel subscription state
from within the handler.
*/
type subscription struct {
	reader *conn
	active bool
}

func newChannel(h *handler, name string) *channel {
	ch := &channel{name: name, owner: h, subscribers: make(subscriberMap)}
	ch.subscribe, ch.broadcast = make(chan subscription), make(chan broadcaster)
	go ch.hub()
	return ch
}

// Channel's hub manages subscriptions and broacdasts messages to all subscribers.
func (ch *channel) hub() {
	for {
		select {
		case s := <-ch.subscribe:
			ch.subscribers[s.reader] = 0, s.active
		case b := <-ch.broadcast:
			for reader := range ch.subscribers {
				b(reader)
			}
		}
	}
}

/*
Handler handlers all incoming requestes using defined protocol. Handler also
manages all registered channels.

Trivial custom handler:

    type MyHandler struct {
        channels map[string]
    }

    func (*h MyHandler) Register(id interface{}) {
        // initialize your handler here...
    }
*/
type Handler interface {
	Register(s *Server, id interface{}) (websocket.Handler, os.Error)
}
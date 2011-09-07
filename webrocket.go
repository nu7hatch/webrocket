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

// NamedEvent is an raw event with specified name.
type NamedEvent struct {
	Event string
}

// ChanneledEvent is an raw event associated with specified channel.
type ChanneledEvent struct {
	Event   string
	Channel string
}

/*
DataEvent is an raw event with data attributes, associated with
specified channel.
*/
type DataEvent struct {
	Event   string
	Channel string
	Data    DataMap
}

// EventError is response payload containig error string.
type ErrorEvent struct {
	Error string
}

// DataMap contains decoded data attributes of payload message.
type DataMap map[string]string

// channelMap is a map of channels.
type channelMap map[string]*channel

// readerMap contains sockets subscribing given channel.
type readerMap map[*websocket.Conn]int

// handlerMap is a map of resource handlers.
type handlerMap map[string]Handler

var (
	invalidChannelErr     = ErrorEvent{"invalid_channel"}
	invalidEventFormatErr = ErrorEvent{"invalid_event_format"}
	invalidAuthDataErr    = ErrorEvent{"invalid_auth_data"}
	accessDeniedErr       = ErrorEvent{"access_denied"}
	notAuthenticatedErr   = ErrorEvent{"not_authenticated"}
)

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

/*
channel keeps information about specified channel and it's subscriptions.
It's hub is used to broadcast messages.
*/
type channel struct {
	name      string
	owner     *handler
	readers   readerMap
	subscribe chan subscription
	broadcast chan broadcaster
}

// broadcaster is a function for spreading messages to all chanel's readers. 
type broadcaster func(ws *websocket.Conn)

/*
subscription struct is used to modify channel subscription state
from within the handler.
*/
type subscription struct {
	reader *websocket.Conn
	active bool
}

func newChannel(h *handler, name string) *channel {
	ch := &channel{name: name, owner: h, readers: make(readerMap)}
	ch.subscribe, ch.broadcast = make(chan subscription), make(chan broadcaster)
	go ch.hub()
	return ch
}

// Channel's hub manages subscriptions and broacdasts messages to all readers.
func (ch *channel) hub() {
	for {
		select {
		case s := <-ch.subscribe:
			ch.readers[s.reader] = 0, s.active
		case b := <-ch.broadcast:
			for reader := range ch.readers {
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

// Default handler, with various message codecs support.
type handler struct {
	Codec      websocket.Codec
	Secret     string
	Log        *log.Logger
	server     *Server
	handler    websocket.Handler
	path       string
	registered bool
	channels   channelMap
	logins     readerMap
}

/*
Creates new handler based on specified websocket's codec. Here's an trivial example:

     server := webrocket.NewServer("localhost:8080")
     handler := webrocket.NewHandler(websocket.JSON)
     server.Handle("/echo", handler)
     server.ListenAndServe()
*/
func NewHandler(codec websocket.Codec) *handler {
	return &handler{Codec: codec}
}

/*
Register initializes new handle under specified id (in this case an id is query path),
and returns valid websocket.Handler clojure to handle incoming messages.
*/
func (h *handler) Register(s *Server, id interface{}) (websocket.Handler, os.Error) {
	if h.registered {
		return nil, os.NewError("Handler already registered")
	}
	if h.Log == nil {
		h.Log = log.New(os.Stderr, id.(string)+" : ", log.LstdFlags)
	}
	h.server = s
	h.path = id.(string)
	h.handler = func(ws *websocket.Conn) { h.eventLoop(ws) }
	h.channels = make(channelMap)
	h.logins = make(readerMap)
	h.registered = true
	s.Log.Printf("Registered handler: %s\n", h.path)
	return h.handler, nil
}

func (h *handler) eventLoop(ws *websocket.Conn) {
	err := h.onOpen(ws)
	if err == nil {
		for {
			var e DataEvent
			err = h.receive(ws, &e)
			if err == nil {
				switch e.Event {
				case "ok":
					continue
				case "subscribe":
					h.onSubscribe(ws, &e)
				case "unsubscribe":
					h.onUnsubscribe(ws, &e)
				case "authenticate":
					h.onAuthenticate(ws, &e)
				default:
					h.onEvent(ws, &e)
				}
			}
			if err == os.EOF {
				break
			}
		}
	}
	h.onClose(ws)
}

func (h *handler) loggedIn(ws *websocket.Conn) bool {
	_, ok := h.logins[ws]
	return ok
}

func (h *handler) receive(ws *websocket.Conn, e interface{}) os.Error {
	err := h.Codec.Receive(ws, e)
	if err != nil && err != os.EOF {
		h.Log.Printf("Receive error: %s\n", err.String())
		h.send(ws, invalidEventFormatErr)
	}
	return err
}

func (h *handler) send(ws *websocket.Conn, e interface{}) os.Error {
	err := h.Codec.Send(ws, e)
	if err != nil {
		h.Log.Printf("Send error: %s\n", err.String())
	}
	return err
}

func (h *handler) onOpen(ws *websocket.Conn) os.Error {
	h.Log.Printf("Connected\n")
	return nil
}

func (h *handler) onClose(ws *websocket.Conn) os.Error {
	h.Log.Printf("Disconnected\n")
	return nil
}

func (h *handler) onSubscribe(ws *websocket.Conn, e *DataEvent) os.Error {
	name := e.Channel
	if len(name) == 0 {
		err := os.NewError("invalid channel: " + name)
		h.Log.Printf("Subscribtion error: %s\n", err.String())
		h.send(ws, invalidChannelErr)
		return err
	}
	ch, ok := h.channels[name]
	if !ok {
		ch = newChannel(h, name)
		h.channels[name] = ch
	}
	h.Log.Printf("Subscribed: %s\n", name)
	h.send(ws, NamedEvent{"ok"})
	ch.subscribe <- subscription{ws, true}
	return nil
}

func (h *handler) onUnsubscribe(ws *websocket.Conn, e *DataEvent) os.Error {
	name := e.Channel
	if ch, ok := h.channels[name]; ok {
		ch.subscribe <- subscription{ws, false}
		h.send(ws, ChanneledEvent{"unsubscribed", name})
		h.Log.Printf("Unsubscribed: %s\n", name, name)
	}
	return nil
}

func (h *handler) onAuthenticate(ws *websocket.Conn, e *DataEvent) os.Error {
	if h.loggedIn(ws) {
		return nil
	}
	secret, ok := e.Data["secret"]
	if h.Secret != "" && !(ok && h.Secret == secret) {
		h.send(ws, notAuthenticatedErr)
		h.Log.Printf("Authentication failed\n")
		return os.NewError("not authenticated")
	}
	h.Log.Printf("Authenticated\n")
	h.send(ws, NamedEvent{"ok"})
	h.logins[ws] = 0, true
	return nil
}

func (h *handler) onEvent(ws *websocket.Conn, e *DataEvent) os.Error {
	name := e.Channel
	ch, ok := h.channels[name]
	if !ok {
		err := os.NewError("invalid channel: " + name)
		h.Log.Printf("Event error: %s\n", err.String())
		h.send(ws, invalidChannelErr)
		return err
	}
	if !h.loggedIn(ws) {
		err := os.NewError("not authorized to publish on this channel")
		h.Log.Printf("[=> %s] Event error: %s\n", name, err.String())
		h.send(ws, accessDeniedErr)
		return err
	}
	ch.broadcast <- func(ws *websocket.Conn) {
		if ws != nil {
			err := h.send(ws, e)
			if err != nil {
				h.Log.Printf("=> %s / Event error: %s\n", name, err.String())
			}
		}
	}
	h.send(ws, NamedEvent{"ok"})
	h.Log.Printf("[=> %s] Broadcasted: %s\n", name, e)
	return nil
}

// Creates new handler basd on the default JSON protocol.
func NewJSONHandler() *handler {
	return NewHandler(websocket.JSON)
}

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
type handlerMap map[string]*Handler

var (
	invalidChannelErr     = ErrorEvent{"INVALID_CHANNEL"}
	invalidEventFormatErr = ErrorEvent{"INVALID_EVENT_FORMAT"}
	accessDeniedErr       = ErrorEvent{"ACCESS_DENIED"}
)

// Server defines parameters for running an WebSocket server.
type Server struct {
	http.Server
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
         s.Handle("/echo", rocket.JSONHandler())
         s.ListenAndServe()
    }
*/
func NewServer(addr string) *Server {
	s := new(Server)
	s.Addr, s.Handler = addr, http.NewServeMux()
	s.handlers = make(handlerMap)
	return s
}

/*
Registers payload handler under specified path. Handler have to implement
communication protocol callbacks.
*/
func (s *Server) Handle(path string, handler *Handler) {
	if !handler.registered {
		handler.register(path)
		s.Handler.(*http.ServeMux).Handle(path, handler.handler)
		s.handlers[path] = handler
	}
}

/*
Listens on the TCP network address srv.Addr and handles requests on incoming
websocket connections.
*/
func (s *Server) ListenAndServe() os.Error {
	log.Printf("About to listen on %s\n", s.Addr)
	err := s.Server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server startup error: %s\n", err.String())
	}
	return err
}

/*
Listens on the TCP network address srv.Addr and handles requests on incoming TLS
websocket connections.
*/
func (s *Server) ListenAndServeTLS(certFile, keyFile string) os.Error {
	log.Printf("About to listen on %s", s.Addr)
	err := s.Server.ListenAndServeTLS(certFile, keyFile)
	if (err != nil) {
		log.Fatalf("Secured server startup error: %s\n", err.String())
	}
	return err
}

/*
channel keeps information about specified channel and it's subscriptions.
It's hub is used to broadcast messages.
*/
type channel struct {
	name      string
	owner     *Handler
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

func newChannel(h *Handler, name string) *channel {
	ch := &channel{name: name, owner: h, readers: make(readerMap)}
	ch.subscribe, ch.broadcast = make(chan subscription), make(chan broadcaster)
	go ch.hub()
	return ch
}

func (ch *channel) hub() {
	for {
		select {
		case s := <-ch.subscribe:
			ch.readers[s.reader] = 0, s.active
		case b := <-ch.broadcast:
			for reader, _ := range ch.readers {
				b(reader)
			}
		}
	}
}

/*
Handler handlers all incoming requestes using defined protocol. Handler also
manages all registered channels.
*/
type Handler struct {
	Codec      websocket.Codec
	handler    websocket.Handler
	path       string
	registered bool
	connected  bool
	channels   channelMap
}

/*
Creates new handler based on specified websocket's codec. Here's an trivial example:

     server := webrocket.NewServer("localhost:8080")
     handler := webrocket.NewHandler(websocket.JSON)
     server.Handle("/echo", handler)
     server.ListenAndServe()
*/
func NewHandler(codec websocket.Codec) *Handler {
	return &Handler{Codec: codec}
}

func (h *Handler) register(path string) {
	h.path = path
	h.handler = func(ws *websocket.Conn) { h.eventLoop(ws) }
	h.channels = make(channelMap)
	h.registered = true
}

func (h *Handler) eventLoop(ws *websocket.Conn) {
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

func (h *Handler) receive(ws *websocket.Conn, e interface{}) os.Error {
	err := h.Codec.Receive(ws, e)
	if err != nil && err != os.EOF {
		log.Printf("Receive error: %s\n", err.String())
		h.send(ws, invalidEventFormatErr)
	}
	return err
}

func (h *Handler) send(ws *websocket.Conn, e interface{}) os.Error {
	err := h.Codec.Send(ws, e)
	if err != nil {
		log.Printf("Send error: %s\n", err.String())
	}
	return err
}

func (h *Handler) onOpen(ws *websocket.Conn) os.Error {
	err := h.send(ws, NamedEvent{"connected"})
	if err != nil {
		return err
	}
	log.Printf("[%s] Connected: %s\n", h.path, "")// ws.RemoteAddr())
	return nil
}

func (h *Handler) onClose(ws *websocket.Conn) os.Error {
	log.Printf("[%s] Connection closed: %s\n", h.path, "") //ws.RemoteAddr())
	return nil
}


func (h *Handler) onSubscribe(ws *websocket.Conn, e *DataEvent) os.Error {
	name := e.Channel
	if len(name) == 0 {
		err := os.NewError("invalid channel: " + name)
		log.Printf("[%s] Subscribtion error: %s\n", h.path, err.String())
		h.send(ws, invalidChannelErr)
		return err
	}
	ch, ok := h.channels[name]
	if !ok {
		ch = newChannel(h, name)
		h.channels[name] = ch
	}
	ch.subscribe <- subscription{ws, true}
	h.send(ws, ChanneledEvent{"subscribed", name})
	log.Printf("[%s => %s] Subscribed: %s\n", h.path, name, name)
	return nil
}

func (h* Handler) onUnsubscribe(ws *websocket.Conn, e *DataEvent) os.Error {
	name := e.Channel
	if ch, ok := h.channels[name]; ok {
		ch.subscribe <- subscription{ws, false}
		h.send(ws, ChanneledEvent{"unsubscribed", name})
		log.Printf("[%s => %s] Unsubscribed: %s\n", h.path, name, name)
	}
	return nil
}

func (h* Handler) onEvent(ws *websocket.Conn, e *DataEvent) os.Error {
	name := e.Channel
	ch, ok := h.channels[name]
	if !ok {
		err := os.NewError("invalid channel: " + name)
		log.Printf("[%s => %s] Event error: %s\n", h.path, name, err.String())
		h.send(ws, invalidChannelErr)
		return err
	}
	ch.broadcast <- func(ws *websocket.Conn) {
		if ws != nil {
			err := h.send(ws, e)
			if err != nil {
				log.Printf("[%s => %s] Event error: %s\n", h.path, name, err.String())
			}
		}
	}
	h.send(ws, NamedEvent{"ok"})
	log.Printf("[%s => %s] Broadcasted: %s\n", h.path, name, e)
	return nil
}

// Creates new handler basd on the default JSON protocol.
func JSONHandler() *Handler {
	return NewHandler(websocket.JSON) 
}
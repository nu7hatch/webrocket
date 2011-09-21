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

// Storage for credentials setup.
type Credentials struct {
	ReadOnly  string
	ReadWrite string
}

// Default handler, with various message codecs support.
type handler struct {
	Codec      websocket.Codec
	Secrets    Credentials
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

type Payload map[string]interface{}
type Data    map[string]interface{}

func (p *Payload) Event() (string, os.Error) {
	for k, _ := range *p {
		return k, nil
	}
	return "", os.NewError("invalid event")
}

func (p *Payload) Data() (*Data, os.Error) {
	for _, v := range *p {
		var d Data
		val, ok := v.(map[string]interface{})
		if ok {
			d = val
			return &d, nil
		}
		_, ok = v.(bool)
		if ok {
			return &d, nil
		}
	}
	return nil, os.NewError("invalid data")
}

var (
	Ok                 Payload = Payload{"ok": true}
	InvalidData        Payload = Payload{"err": "invalid_data"}
	InvalidCredentials Payload = Payload{"err": "invalid_credentials"}
	InvalidChannelName Payload = Payload{"err": "invalid_channel_name"}
	InvalidChannel     Payload = Payload{"err": "invalid_channel"}
	AccessDenied       Payload = Payload{"err": "access_denied"}
)

const (
	ReadOnlyAccess  = "read-only"
	ReadWriteAccess = "read-write"
)

var AccessCodes map[string]int = map[string]int{
	ReadOnlyAccess: 1,
	ReadWriteAccess: 2,
}
	
func (h *handler) eventLoop(ws *websocket.Conn) {
	h.onOpen(ws)
	for {
		var recv Payload
		err = h.Codec.Receive(ws, &recv)
		if err != nil {
			if err == os.EOF {
				break
			}
			h.onError(ws, err)
			continue
		}
		event, err := recv.Event()
		if err != nil {
			h.onError(ws, err)
			continue
		}
		data, err := recv.Data()
		if err != nil {
			h.onError(ws, err)
			continue
		}
		ok := h.dispatch(ws, event, data)
		if !ok {
			break
		}
	}
	h.onClose(ws)
}

func (h *handler) dispatch(ws *websocket.Conn, event string, data *Data) bool {
	switch event {
	case "auth":
		h.onAuthenticate(ws, data)
	case "subscribe":
		h.onSubscribe(ws, data)
	case "unsubscribe":
		h.onUnsubscribe(ws, data)
	case "publish":
		h.onPublish(ws, data)
	case "logout":
		h.onLogout(ws)
	case "disconnect":
		h.onDisconnect(ws)
		return false
	}
	return true
}


func (h *handler) send(ws *websocket.Conn, data interface{}) os.Error {
	err := h.Codec.Send(ws, data)
	if err != nil {
		h.Log.Printf("Error: %s\n", err.String())
	}
	return err
}

func (h *handler) assertAccess(ws *websocket.Conn, access string) bool {
	code, ok := h.logins[ws]
	if !ok || code < AccessCodes[access] {
		h.Log.Printf("Error: access denied\n")
		h.send(ws, AccessDenied)
		return false
	}
	return true
}

func (h *handler) secretFor(access string) (string, os.Error) {
	if access == ReadOnlyAccess {
		return h.Secrets.ReadOnly, nil
	}
	if access == ReadWriteAccess {
		return h.Secrets.ReadWrite, nil
	}
	return "", os.NewError("invalid access method\n")
}

func (h *handler) onError(ws *websocket.Conn, err os.Error) {
	h.Log.Printf("Error: %s\n", err.String())
	h.send(ws, InvalidData)
}

func (h *handler) onOpen(ws *websocket.Conn) {
	h.Log.Printf("Connected\n")
}

func (h *handler) onClose(ws *websocket.Conn) {
	h.Log.Printf("Disconnected\n")
}

func (h *handler) onDisconnect(ws *websocket.Conn) {
	h.send(ws, Ok)
	ws.Close()
}

func (h *handler) onAuthenticate(ws *websocket.Conn, data *Data) {
	access, ok := (*data)["access"]
	if !ok {
		h.onError(ws, os.NewError("invalid auth data"))
		return
	}
	secret, ok := (*data)["secret"]
	if !ok {
		h.onError(ws, os.NewError("invalid auth data"))
		return
	}
	accessId, ok := access.(string)
	if !ok {
		h.onError(ws, os.NewError("invalid auth data"))
		return
	}
	validSecret, err := h.secretFor(accessId)
	if err != nil {
		h.onError(ws, err)
		return
	}
	if validSecret != "" && validSecret != secret {
		h.logins[ws] = 0, false
		h.send(ws, InvalidCredentials)
		h.Log.Printf("Authentication failed\n")
		return
	}
	h.Log.Printf("Authenticated (%s access)\n", accessId)
	h.logins[ws] = AccessCodes[accessId], true
	h.send(ws, Ok)
}

func (h *handler) onLogout(ws *websocket.Conn) {
	h.Log.Printf("Logged out\n")
	h.logins[ws] = 0, false
	h.send(ws, Ok)
}

func (h *handler) onSubscribe(ws *websocket.Conn, data *Data) {
	ok := h.assertAccess(ws, ReadOnlyAccess)
	if !ok {
		return
	}
	channel, ok := (*data)["channel"]
	if !ok {
		h.onError(ws, os.NewError("invalid subscribe data"))
		return
	}
	name, ok := channel.(string)
	if !ok {
		h.onError(ws, os.NewError("invalid unsubscribe data"))
		return
	}
	if len(name) == 0 {
		err := os.NewError("empty channel name")
		h.Log.Printf("Error: %s\n", err.String())
		h.send(ws, InvalidChannelName)
		return
	}
	ch, ok := h.channels[name]
	if !ok {
		ch = newChannel(h, name)
		h.channels[name] = ch
	}
	h.Log.Printf("Subscribed: %s\n", name)
	ch.subscribe <- subscription{ws, true}
	h.send(ws, Ok)
}

func (h *handler) onUnsubscribe(ws *websocket.Conn, data *Data) {
	channel, ok := (*data)["channel"]
	if !ok {
		h.onError(ws, os.NewError("invalid unsubscribe data"))
		return
	}
	name, ok := channel.(string)
	if !ok {
		h.onError(ws, os.NewError("invalid unsubscribe data"))
		return
	}
	ch, ok := h.channels[name]
	if ok {
		h.Log.Printf("Unsubscribed: %s\n", name)
		ch.subscribe <- subscription{ws, false}
	}
	h.send(ws, Ok)
}

func (h *handler) onPublish(ws *websocket.Conn, data *Data) {
	ok := h.assertAccess(ws, ReadWriteAccess)
	if !ok {
		return
	}
	_, ok = (*data)["event"]
	if !ok {
		h.onError(ws, os.NewError("invalid publish data"))
		return
	}
	channel, ok := (*data)["channel"]
	if !ok {
		h.onError(ws, os.NewError("invalid publish data"))
		return
	}	
	name, ok := channel.(string)
	if !ok {
		h.onError(ws, os.NewError("invalid publish data"))
		return
	}
	ch, ok := h.channels[name]
	if !ok {
		err := os.NewError("invalid channel: " + name)
		h.Log.Printf("Error: %s\n", err.String())
		h.send(ws, InvalidChannel)
		return
	}
	ch.broadcast <- func (reader *websocket.Conn) {
		if reader != nil {
			h.send(reader, *data)
		}
	}
	h.Log.Printf("[=> %s] Broadcasted: %s\n", name, *data)
	h.send(ws, Ok)
}

// Creates new handler basd on the default JSON protocol.
func NewJSONHandler() *handler {
	return NewHandler(websocket.JSON)
}

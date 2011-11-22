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
	"container/list"
	"crypto/sha1"
	"fmt"
)

// Wrapper for standard websocket.Conn structure.
type conn struct {
	*websocket.Conn
	token string
}

// generateUniqueToken creates unique token using system `/dev/urandom`.
func generateUniqueToken() string {
	f, _ := os.OpenFile("/dev/urandom", os.O_RDONLY, 0) 
	b := make([]byte, 16) 
	f.Read(b) 
	f.Close() 
	token := sha1.New()
	token.Write(b)
	return fmt.Sprintf("%x", token.Sum())
}

/*
wrapConn wraps standard websocket connection object into one adjusted for
webrocker server funcionalities.
*/
func wrapConn(ws *websocket.Conn) *conn {
	return &conn{Conn: ws, token: generateUniqueToken()}
}

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

// readerMap contains sockets subscribing given channel.
type readerMap map[*conn]int

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

// channelMap is a map of channels.
type channelMap map[string]*channel

// connectionMap is a map of active connections.
type connectionMap map[string]*conn

// broadcaster is a function for spreading messages to all chanel's readers. 
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
Payload is an general strucutre for all sent event messages.

Simple examples how to create new event message:

    Payload SimpleMessage  = Payload("hello": "world")
    Payload ComplexMessage = Payload("hello": Data{"foo": "bar"})
*/
type Payload map[string]interface{}

// Data is an general structure for all received event messages.
type Data map[string]interface{}

// Returns name of event represented by this payload.
func (p *Payload) Event() (string, os.Error) {
	for k := range *p {
		return k, nil
	}
	return "", os.NewError("No event specified")
}

// Returns data contained by this payload.
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
	return nil, os.NewError("Invalid format of the data")
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

// Access control defaults.
var AccessCodes map[string]int = map[string]int{
	ReadOnlyAccess:  1,
	ReadWriteAccess: 2,
}

// Access control constants.
const (
	ReadOnlyAccess  = "read-only"
	ReadWriteAccess = "read-write"
)

// Predefined reply for successfull authentication.
func Authenticated(accessType string) *Payload {
	return &Payload{"authenticated": accessType}
}

// Predefined reply for successfull subscription.
func Subscribed(channelName string) *Payload {
	return &Payload{"subscribed": channelName}
}

// Predefined reply for successfull unsubscription.
func Unsubscribed(channelName string) *Payload {
	return &Payload{"unsubscribed": channelName}
}

// Predefined reply for successfull broadcast.
func Broadcasted(channelName string) *Payload {
	return &Payload{"broadcasted": channelName}
}

// Other predefined payloads.
var (
	LoggedOut Payload = Payload{"loggedOut": true}
)

// Error payloads.
var (
	InvalidPayload       Payload = Payload{"err": "INVALID_PAYLOAD"}
	InvalidCredentials   Payload = Payload{"err": "INVALID_CREDENTIALS"}
	InvalidChannel       Payload = Payload{"err": "INVALID_CHANNEL"}
	AccessDenied         Payload = Payload{"err": "ACCESS_DENIED"}
)

// Default handler, with various message codecs support.
type handler struct {
	Codec         websocket.Codec
	Secrets       Credentials
	Log           *log.Logger
	server        *Server
	handler       websocket.Handler
	path          string
	registered    bool
	connections   connectionMap
	channels      channelMap
	logins        readerMap
	subscriptions *list.List
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
	h.handler = func(ws *websocket.Conn) {
		wrapped := wrapConn(ws)
		h.eventLoop(wrapped)
	}
	h.connections = make(connectionMap)
	h.channels = make(channelMap)
	h.logins = make(readerMap)
	h.subscriptions = list.New()
	h.registered = true
	s.Log.Printf("Registered handler: %s\n", h.path)
	return h.handler, nil
}

func (h *handler) eventLoop(ws *conn) {
	h.onOpen(ws)
	for {
		var recv Payload
		err := h.Codec.Receive(ws.Conn, &recv)
		if err != nil {
			if err == os.EOF {
				break
			}
			h.onError(ws, InvalidPayload, err)
			continue
		}
		event, err := recv.Event()
		if err != nil {
			h.onError(ws, InvalidPayload, err)
			continue
		}
		data, err := recv.Data()
		if err != nil {
			h.onError(ws, InvalidPayload, err)
			continue
		}
		ok := h.dispatch(ws, event, data)
		if !ok {
			break
		}
	}
	h.onClose(ws)
}

func (h *handler) dispatch(ws *conn, event string, data *Data) bool {
	switch event {
	case "authenticate":
		h.onAuthenticate(ws, data)
	case "subscribe":
		h.onSubscribe(ws, data)
	case "unsubscribe":
		h.onUnsubscribe(ws, data)
	case "broadcast":
		h.onBroadcast(ws, data)
	case "logout":
		h.onLogout(ws)
	case "disconnect":
		h.onDisconnect(ws)
		return false
	}
	return true
}

func (h *handler) send(ws *conn, data interface{}) os.Error {
	err := h.Codec.Send(ws.Conn, data)
	if err != nil {
		h.Log.Printf("[%s] \033[35m~~> %s\033[0m\n", ws.token, err.String())
	}
	return err
}

func (h *handler) assertAccess(action string, ws *conn, access string) bool {
	code, ok := h.logins[ws]
	if !ok || code < AccessCodes[access] {
		h.Log.Printf("[%s] \033[33m[-> %s] Access denied\033[0m\n", ws.token, action)
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
	return "", os.NewError("Invalid access method\n")
}

/*
onError is a helper for dealing with failures caused mainly by invalid payload format,
or other message problems.
*/
func (h *handler) onError(ws *conn, payload Payload, err os.Error) {
	errName := payload["err"]
	h.Log.Printf("[%s] \033[31m[<~ %s] %s\033[0m\n", ws.token, errName, err.String())
	h.send(ws, payload)
}

/*
onMinorError is a helper for dealing with small errors, like failed authentication,
access denied errors etc.
*/
func (h *handler) onMinorError(ws *conn, payload Payload, err os.Error) {
	errName := payload["err"]
	h.Log.Printf("[%s] \033[33m[<- %s] %s\033[0m\n", ws.token, errName, err.String())
	h.send(ws, payload)
}

/*
onOpen handles registration of the new connection in the system. Each new connection
has assigned SHA1 token to easily distinguish them from the others.
*/
func (h *handler) onOpen(ws *conn) {
	h.connections[ws.token] = ws
	h.Log.Printf("[%s] \033[34m~~> Connected\033[0m\n", ws.token)
}

func (h *handler) onClose(ws *conn) {
	h.Log.Printf("[%s] \033[34m<~~ Disconnected\033[0m\n", ws.token)
}

// onClose handles safe disconnection requested by the client.
func (h *handler) onDisconnect(ws *conn) {
	ws.Close()
}

// onAuthenticate manages session authentication for the connected client.
func (h *handler) onAuthenticate(ws *conn, data *Data) {
	access, ok := (*data)["access"]
	if !ok {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Missing access type: %s", *data)))
		return
	}
	secret, ok := (*data)["secret"]
	if !ok {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Missing secret: %s", *data)))
		return
	}
	accessId, ok := access.(string)
	if !ok {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Invalid access type data: %s", *data)))
		return
	}
	validSecret, err := h.secretFor(accessId)
	if err != nil {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Invalid access type: %s", *data)))
		return
	}
	if validSecret != "" && validSecret != secret {
		h.onMinorError(ws, InvalidCredentials, os.NewError("Authentication failed"))
		h.logins[ws] = 0, false
		return
	}
	h.Log.Printf("[%s] \033[36m~~> Authenticated (%s access)\033[0m\n", ws.token, accessId)
	h.logins[ws] = AccessCodes[accessId], true
	h.send(ws, Authenticated(accessId))
}

// onLogout finishes current session and unsubscribes all channels subscribed by the client. 
func (h *handler) onLogout(ws *conn) {
	h.Log.Printf("[%s] \033[34m<~~ Logged out\033[0m\n", ws.token)
	h.logins[ws] = 0, false
	for e := h.subscriptions.Front(); e != nil; e = e.Next() {
		ch := e.Value.(*channel)
		ch.subscribe <- subscription{ws, false}
	}
	h.send(ws, LoggedOut)
}

// onSubscribe handlers subscription of the specified channel.
func (h *handler) onSubscribe(ws *conn, data *Data) {
	ok := h.assertAccess("SUBSCRIBE", ws, ReadOnlyAccess)
	if !ok {
		return
	}
	chanName, ok := (*data)["channel"]
	if !ok {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Missing channel name: %s", *data)))
		return
	}
	name, ok := chanName.(string)
	if !ok {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Invalid channel name: %s", *data)))
		return
	}
	if len(name) == 0 {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Invalid channel name: %s", *data)))
		return
	}
	ch, ok := h.channels[name]
	if !ok {
		ch = newChannel(h, name)
		h.channels[name] = ch
	}
	h.Log.Printf("[%s] \033[32m[-> SUBSCRIBE ~ %s] Channel subscribed\033[0m\n", ws.token, name)
	ch.subscribe <- subscription{ws, true}
	h.subscriptions.PushBack(ch)
	h.send(ws, Subscribed(name))
}

// onUnsubscribe handles unsubscribing of the specified channel.
func (h *handler) onUnsubscribe(ws *conn, data *Data) {
	chanName, ok := (*data)["channel"]
	if !ok {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Missing channel name: %s", *data)))
		return
	}
	name, ok := chanName.(string)
	if !ok {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Invalid channel name: %s", *data)))
		return
	}
	if len(name) == 0 {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Invalid channel name: %s", *data)))
		return
	}
	ch, ok := h.channels[name]
	if ok {
		ch.subscribe <- subscription{ws, false}
		for e := h.subscriptions.Front(); e != nil; e = e.Next() {
			cmp := e.Value.(*channel)
			if ch == cmp {
				h.subscriptions.Remove(e)
			}
		}
		h.Log.Printf("[%s] \033[32m[-> UNSUBSCRIBE ~ %s] Channel unsubscribed\033[0m\n", ws.token, name)
	}
	h.send(ws, Unsubscribed(name))
}

/*
onBroadcasts handles message from the current connection and spreads it out across all
clients subscribing specified channel.
*/
func (h *handler) onBroadcast(ws *conn, data *Data) {
	ok := h.assertAccess("BROADCAST", ws, ReadWriteAccess)
	if !ok {
		return
	}
	_, ok = (*data)["event"]
	if !ok {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Missing event name: %s", *data)))
		return
	}
	channel, ok := (*data)["channel"]
	if !ok {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Missing channel name: %s", *data)))
		return
	}
	name, ok := channel.(string)
	if !ok {
		h.onError(ws, InvalidPayload, os.NewError(fmt.Sprintf("Invalid channel name: %s", *data)))
		return
	}
	ch, ok := h.channels[name]
	if !ok {
		h.onMinorError(ws, InvalidChannel, os.NewError(fmt.Sprintf("Channel does not exist: %s", *data)))
		return
	}
	ch.broadcast <- func(reader *conn) {
		if reader != nil {
			h.send(reader, *data)
		}
	}
	h.Log.Printf("[%s] \033[32m[=> BROADCAST ~ %s] Broadcasted: %s\033[0m\n", ws.token, name, *data)
	h.send(ws, Broadcasted(name))
}

// Creates new handler basd on the default JSON protocol.
func NewJSONHandler() *handler {
	return NewHandler(websocket.JSON)
}

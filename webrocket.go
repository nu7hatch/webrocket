package webrocket

import (
	"websocket"
	"http"
	"log"
	"os"
	"container/vector"
	)

// Decoded data attributes.
type DataMap map[string]string

// Raw named event.
type NamedEvent struct {
	Event string
}

// Raw event associated with specified channel.
type ChanneledEvent struct {
	Event   string
	Channel string
}

// Raw event with data attributes, associated with specified channel. 
type DataEvent struct {
	Event   string
	Channel string
	Data    DataMap
}

// Response payload containig error string.
type ErrorEvent struct {
	Error string
}

// Predefined error payloads.
var (
	InvalidChannelErr     = ErrorEvent{"INVALID_CHANNEL"}
	InvalidEventFormatErr = ErrorEvent{"INVALID_EVENT_FORMAT"}
	AccessDeniedErr       = ErrorEvent{"ACCESS_DENIED"}
)

// A Server defines parameters for running an WebSocket server.
type Server struct {
	http.Server
	
	handlers HandlerMap
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
	s.handlers = make(HandlerMap)
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
Channel keeps information about subscribeing connections and broadcasts information
between them.
*/
type Channel struct {
	Name        string

	owner       *Handler
	subscribers vector.Vector
}

// Map of registered channels.
type ChannelMap map[string]*Channel

func newChannel(h *Handler, name string) *Channel {
	ch := &Channel{Name: name, owner: h}
	return ch
}

func (ch *Channel) subscriberId(ws *websocket.Conn) (n int) {
	for n = 0; n < ch.subscribers.Len(); n += 1 {
		s := ch.subscribers.At(n).(*websocket.Conn)
		if ws == s {
			return n
		}
	}
	return -1
}

// Adds given connection to subscribers of this channel. 
func (ch *Channel) Subscribe(ws *websocket.Conn) {
	if ch.subscriberId(ws) >= 0 {
		return 
	}
	ch.subscribers.Push(ws)
}

// Removes given connection from the list of subscribers.
func (ch *Channel) Unsubscribe(ws *websocket.Conn) {
	n := ch.subscriberId(ws)
	if n >= 0 {
		ch.subscribers.Delete(n)
	}
}

/*
Handler handlers all incoming requestes using defined protocol. Handler also
manages all registered channels.
*/
type Handler struct {
	Protocol   *Protocol

	path       string
	registered bool
	channels   ChannelMap
	handler    websocket.Handler
}

// Map of defined handlers.
type HandlerMap map[string]*Handler

func newHandler(proto Protocol) *Handler {
	return &Handler{Protocol: &proto}
}

func (h *Handler) register(path string) {
	h.path = path
	h.handler = func(ws *websocket.Conn) { h.eventLoop(ws) }
	h.channels = make(ChannelMap)
	h.registered = true
}

func (h *Handler) eventLoop(ws *websocket.Conn) {
	err := h.Protocol.OnConnect(h, ws, nil)
	if (err == nil) {
		for {
			var e DataEvent
			err = h.Protocol.Receive(h, ws, &e)
			if err == nil {
				switch e.Event {
				case "ok":
					continue
				case "subscribe":
					h.Protocol.OnSubscribe(h, ws, &e)
				case "unsubscribe":
					h.Protocol.OnUnsubscribe(h, ws, &e)
				default:
					h.Protocol.OnMessage(h, ws, &e)
				}
			}
		}
	}
	h.Protocol.OnClose(h, ws, nil)
}

/*
Protocol callback method. Some methods may ignore second attribute.
Each protocol method should be defined using this template.
*/
type Callback func(h *Handler, ws *websocket.Conn, e *DataEvent) os.Error

/*
Protocol template allows you to define your own handlers at top of
Rocket's infrastructure. You can use it to implement your own protocol
instead of default, JSON-based one.
*/
type Protocol struct {
	Receive        Callback
	OnConnect      Callback
	OnSubscribe    Callback
	OnUnsubscribe  Callback
	OnMessage      Callback
	OnClose        Callback
}

func jsonReceive(h *Handler, ws *websocket.Conn, e *DataEvent) os.Error {
	err := websocket.JSON.Receive(ws, e)
	if err != nil {
		log.Printf("Receive error: %s\n", err.String())
		websocket.JSON.Send(ws, InvalidEventFormatErr)
	}
	return err
}

func jsonOnConnect(h *Handler, ws *websocket.Conn, _ *DataEvent) os.Error {
	err := websocket.JSON.Send(ws, NamedEvent{"connected"})
	if err != nil {
		log.Printf("[%s] Connection error: %s\n", h.path, err.String())
		return err
	}
	log.Printf("[%s] Connected: %s\n", h.path, "")// ws.RemoteAddr())
	return nil
}

func jsonOnSubscribe(h *Handler, ws *websocket.Conn, e *DataEvent) os.Error {
	name := e.Channel
	if len(name) == 0 {
		err := os.NewError("invalid channel: " + name)
		log.Printf("[%s] Subscribtion error: %s\n", h.path, err.String())
		websocket.JSON.Send(ws, InvalidChannelErr)
		return err
	}

	ch, ok := h.channels[name]
	if !ok {
		ch = newChannel(h, name)
		h.channels[name] = ch
	}

	ch.Subscribe(ws)
	websocket.JSON.Send(ws, ChanneledEvent{"subscribed", name})
	log.Printf("[%s => %s] Subscribed: %s\n", h.path, name, name)
	return nil
}

func jsonOnUnsubscribe(h *Handler, ws *websocket.Conn, e *DataEvent) os.Error {
	name := e.Channel
	ch, ok := h.channels[name]
	if !ok {
		err := os.NewError("invalid channel: " + name)
		log.Printf("[%s] Unsubscription error: %s\n", h.path, err.String())
		websocket.JSON.Send(ws, InvalidChannelErr)
		return err
	}

	ch.Unsubscribe(ws)
	websocket.JSON.Send(ws, ChanneledEvent{"unsubscribed", name})
	log.Printf("[%s => %s] Unsubscribed: %s\n", h.path, name, name)
	return nil
}

func jsonOnEvent(h *Handler, ws *websocket.Conn, e *DataEvent) os.Error {
	name := e.Channel
	ch, ok := h.channels[name]
	if !ok {
		err := os.NewError("invalid channel: " + name)
		log.Printf("[%s => %s] Event error: %s\n", h.path, name, err.String())
		websocket.JSON.Send(ws, InvalidChannelErr)
		return err
	}
	ch.subscribers.Do(func (elem interface{}) {
		s, ok := elem.(*websocket.Conn)
		if ok && s != ws && s != nil {
			err := websocket.JSON.Send(s, e)
			if err != nil {
				log.Printf("[%s => %s] Event error: %s\n", h.path, name, err.String())
			}
		}
	})

	websocket.JSON.Send(ws, NamedEvent{"ok"})
	log.Printf("[%s => %s] Broadcasted: %s\n", h.path, name, e)
	return nil
}

func jsonOnClose(h *Handler, ws *websocket.Conn, e *DataEvent) os.Error {
	log.Printf("[%s] Connection closed: %s\n", h.path, "") //ws.RemoteAddr())
	return nil
}

/*
Default Rocket's protocol. Implements JSON data handlers to subscribe channels
and broadcast data between them.
*/
var JSONProtocol = Protocol{jsonReceive, jsonOnConnect, jsonOnSubscribe, jsonOnUnsubscribe, jsonOnEvent, jsonOnClose}

// Creates new handler basd on the default JSON protocol.
func JSONHandler() *Handler {
	return newHandler(JSONProtocol) 
}
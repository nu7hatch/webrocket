package main

import (
	"strconv"
	"websocket"
	"http"
	"log"
	"os"
	)

type DataMap map[string]string

type NamedEvent struct {
	Event   string
}

type ChanneledEvent struct {
	Event   string
	Channel string
}

type DataEvent struct {
	Event   string
	Channel string
	Data    DataMap
}

type ErrorEvent struct {
	Error   string
}

type Channel struct {
	Name     string

	//backend  chan *Event
	ownerHub *Hub
}

func newChannel(hub *Hub, name string) *Channel {
	ch := &Channel{Name: name, ownerHub: hub}
	//ch.backend = make(chan *Event)
	return ch
}

type Hub struct {
	Name       string
	Passphrase string
	Handler    websocket.Handler

	channels   map[string]*Channel
	registered bool
}

func NewHub(name, pass string) *Hub {
	h := &Hub{Name: name, Passphrase: pass}
	h.Handler = func(ws *websocket.Conn) { h.serve(ws) }
	h.channels = make(map[string]*Channel)
	return h
}

func (h *Hub) setup(ws *websocket.Conn) os.Error {
	err := websocket.JSON.Send(ws, NamedEvent{"connected"})
	if err != nil {
		log.Printf("Connection error: %s!\n", err.String())
		return err
	}
	
	//log.Printf("Connected: %s\n", ws.RemoteAddr())
	return nil
}

func (h *Hub) subscribe(ws *websocket.Conn, event *DataEvent) os.Error {
	if len(event.Channel) == 0 {
		err := os.NewError("invalid channel")
		log.Printf("Subscribe error: %s\n", err.String())
		websocket.JSON.Send(ws, ErrorEvent{"INVALID_CHANNEL"})
		return err
	}

	ch, ok := h.channels[event.Channel]
	if !ok {
		ch = newChannel(h, event.Channel)
		h.channels[event.Channel] = ch
	}
	
	err := websocket.JSON.Send(ws, ChanneledEvent{"subscribed", ch.Name})
	if err != nil {
		log.Printf("Subscribe error: %s\n", err.String())
		return err
	}

	log.Printf("Subscribed: %s\n", ch.Name)
	return nil
}

func (h *Hub) unsubscribe(ws *websocket.Conn, event *DataEvent) os.Error {
	ch, ok := h.channels[event.Channel]
	if !ok {
		err := os.NewError("invalid channel")
		log.Printf("Unsubscribe error: %s\n", err.String())
		websocket.JSON.Send(ws, ErrorEvent{"INVALID_CHANNEL"})
		return err
	}
	
	err := websocket.JSON.Send(ws, ChanneledEvent{"unsubscribed", ch.Name})
	if err != nil {
		log.Printf("Unsubscribe error: %s\n", err)
		return err
	}
	
	log.Printf("Unsubscribed %s\n", ch.Name)
	return nil
}

func (h *Hub) serve(ws *websocket.Conn) {
	err := h.setup(ws)
	if err == nil {
		for {
			var event DataEvent
			err = websocket.JSON.Receive(ws, &event)
			
			if err == nil {
				switch event.Event {
				case "__subscribe__":
					h.subscribe(ws, &event)
				case "__unsubscribe__":
					h.unsubscribe(ws, &event)
				default:
					websocket.JSON.Send(ws, ErrorEvent{"INVALID_COMMAND"})
				}
			} else {
				log.Printf("Receive error: %s!\n", err.String())
			}
	    }
	}
}

type Server struct {
	Host string
	Port uint
	
	hubs map[string]*Hub
	backend  *http.Server
}

func NewServer(host string, port uint) *Server {
	s := &Server{Host: host, Port: port}
	s.hubs = make(map[string]*Hub)
	s.backend = &http.Server{Addr: s.url(), Handler: http.NewServeMux()}
	return s
}

func (s *Server) url() string {
	return s.Host + ":" + strconv.Uitoa(s.Port)
}

func (s *Server) RegisterHub(hub *Hub) os.Error {
	if hub.registered {
		return os.NewError("hub already registered")
	}

	handler := websocket.Handler(hub.Handler)
	s.backend.Handler.(*http.ServeMux).Handle(hub.Name, handler)
	s.hubs[hub.Name] = hub
	hub.registered = true

	return nil
}

func (s *Server) CreateHub(name, pass string) os.Error {
	h := NewHub(name, pass)
	return s.RegisterHub(h)
}

func (s *Server) Start() {
	log.Printf("About to listen on ws://%s:%d", s.Host, s.Port)

	err := s.backend.ListenAndServe()
	if err != nil {
		log.Fatalf("Can't setup listener - %s\n", err.String())
	}
}

func main() {
	serv := NewServer("localhost", 8080)
	serv.CreateHub("/echo", "secret")
	serv.Start()
}
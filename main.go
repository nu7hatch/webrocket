package main

import (
	"strconv"
	"websocket"
	"http"
	"log"
	"json"
	"os"
	"bytes"
	)

const (
	HubAlreadyRegisteredErr = 1
	)

type Event struct {
	Event string
	Body  string
	Data  map[string]string
}

type Error struct {
	code int
}

type Channel struct {
	Name     string

	backend  chan *Event
	ownerHub *Hub
}

func newChannel(hub *Hub, name string) *Channel {
	ch := &Channel{Name: name, ownerHub: hub}
	ch.backend = make(chan *Event)
	return ch
}

type Hub struct {
	Name       string
	Passphrase string
	Handler    websocket.Handler

	channels   map[string]*Channel
	registered bool
}

type Conn struct {
	Ws  *websocket.Conn
	
	out *json.Encoder
}

func newConn(ws *websocket.Conn) *Conn {
	c := &Conn{Ws: ws}
	c.out = json.NewEncoder(ws)
	return c
}

func (c *Conn) Read(e *Event) os.Error {
	var buf bytes.Buffer
	msg := make([]byte, 1024)

	for {
		n, err := c.Ws.Read(msg)
		buf.Write(msg[:n])

		if n < 1024 || err == os.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	
	return json.Unmarshal(buf.Bytes(), e)
}

func (c *Conn) Write(e *Event) os.Error {
	return c.out.Encode(e)
}

func NewHub(name, pass string) *Hub {
	h := &Hub{Name: name, Passphrase: pass}
	h.Handler = func(ws *websocket.Conn) { h.serve(ws) }
	h.channels = make(map[string]*Channel)
	return h
}

func (h *Hub) setup(conn *Conn) bool {
	if err := conn.Write(&Event{Event: "connected"}); err != nil {
		log.Printf("Conn error: %s!\n", err.String())
		return false
	}
	
	log.Printf("Connected: %s\n", conn.Ws.Origin)
	return true
}

func (h *Hub) subscribe(conn *Conn, event *Event) bool {
	if name, ok := event.Data["channel"]; ok {
		ch, ok := h.channels[name]
		
		if !ok {
			ch = newChannel(h, name)
			h.channels[name] = ch
		}
	
		log.Printf("Subscribed %s\n", ch.Name)
		err := conn.Write(&Event{Event: "subscribed", Data: map[string]string{"channel": ch.Name}})
		if err == nil { return true }
	}
	return false
}

func (h *Hub) unsubscribe(conn *Conn, event *Event) bool {
	if name, ok := event.Data["channel"]; ok {
		if _, ok := h.channels[name]; ok {
			log.Printf("Unsubscribed %s\n", name)
			err := conn.Write(&Event{Event: "unsubscribed", Data: map[string]string{"channel": name}})
			if err == nil { return true }
		}
	}
	return false
}

func (h *Hub) serve(ws *websocket.Conn) {
	conn := newConn(ws)
	
	if h.setup(conn) {
		for {
			var event Event
			err := conn.Read(&event)

			if err == nil {
				switch event.Event {
				case "__subscribe__":
					h.subscribe(conn, &event)
				case "__unsubscribe__":
					h.unsubscribe(conn, &event)
				}
			} else {
				log.Printf("Event error: %s!\n", err.String())
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

func (s *Server) RegisterHub(hub *Hub) *Error {
	if hub.registered {
		return &Error{HubAlreadyRegisteredErr}
	}

	handler := websocket.Handler(hub.Handler)
	s.backend.Handler.(*http.ServeMux).Handle(hub.Name, handler)
	s.hubs[hub.Name] = hub
	hub.registered = true

	return nil
}

func (s *Server) CreateHub(name, pass string) *Error {
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
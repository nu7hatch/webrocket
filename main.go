package main

import (
	"strconv"
	"websocket"
	"http"
	"log"
	"json"
	)

const (
	ChannelAlreadyRegisteredErr = 1
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
	Name       string
	Passphrase string
	Handler    websocket.Handler

	registered bool
}

func NewChannel(name, pass string) *Channel {
	ch := &Channel{Name: name, Passphrase: pass}
	ch.Handler = func(ws *websocket.Conn) { ch.serve(ws) }
	return ch
}

func (ch *Channel) serve(ws *websocket.Conn) {
	enc := json.NewEncoder(ws)
	enc.Encode(&Event{Event: "connected"})
	
	for {
		var event Event;

		dec := json.NewDecoder(ws)
		err := dec.Decode(&event)

		if err == nil {
			switch event.Event {
			case "subscribe":
				if ch, ok := event.Data["channel"]; ok {
					log.Printf("Subscribing %s\n", ch)
					enc.Encode(&Event{Event: "subscribed", Data: map[string]string{"channel": ch}})
				}
			case "unsubscribe":
				if ch, ok := event.Data["channel"]; ok {
					log.Printf("Unsubscribing %s\n", ch)
					enc.Encode(&Event{Event: "unsubscribed", Data: map[string]string{"channel": ch}})
				}
			}
		} else {
			log.Fatalf("Error: %s\n", err.String())
		}
	}
}

type Server struct {
	Host string
	Port uint
	
	channels map[string]*Channel
	backend  *http.Server
}

func NewServer(host string, port uint) *Server {
	s := &Server{Host: host, Port: port}
	s.channels = make(map[string]*Channel)
	s.backend = &http.Server{Addr: s.url(), Handler: http.NewServeMux()}
	return s
}

func (s *Server) url() string {
	return s.Host + ":" + strconv.Uitoa(s.Port)
}

func (s *Server) RegisterChannel(ch *Channel) *Error {
	if ch.registered {
		return &Error{ChannelAlreadyRegisteredErr}
	}

	handler := websocket.Handler(ch.Handler)
	s.backend.Handler.(*http.ServeMux).Handle(ch.Name, handler)
	s.channels[ch.Name] = ch
	ch.registered = true

	return nil
}

func (s *Server) CreateChannel(name, pass string) *Error {
	ch := NewChannel(name, pass)
	return s.RegisterChannel(ch)
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
	serv.CreateChannel("/echo", "secret")
	serv.Start()
}
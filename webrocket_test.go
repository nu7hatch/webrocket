package webrocket

import (
	"websocket"
	"testing"
	)

var ws *websocket.Conn

func init() {
	go func() {
		server := NewServer(":9771")
		handler := NewHandler(websocket.JSON)
		handler.Secret = "secret"
		server.Handle("/echo", handler)
		server.ListenAndServe()
	}()
}

func sendAndCheck(t *testing.T, data interface{}) {
	err := websocket.JSON.Send(ws, data)
	if err != nil {
		t.Error(err)
	}
}

func checkOkResponse(t *testing.T) {
	var response NamedEvent
	err := websocket.JSON.Receive(ws, &response)
	if err != nil {
		t.Error(err)
	}
	if response.Event != "ok" {
		t.Errorf("Expected `ok` event, `%s` given", response.Event)
	}
}

func TestConnect(t *testing.T) {
	ws_, err := websocket.Dial("ws://localhost:9771/echo", "ws", "http://localhost/")
	ws = ws_
	if err != nil {
		t.Error(err)
	}
}

func TestSubscribe(t *testing.T) {
	subscr := ChanneledEvent{Event: "subscribe", Channel: "test"}
	sendAndCheck(t, subscr)
	checkOkResponse(t)
}

func TestAuthenticate(t *testing.T) {
	auth := DataEvent{Event: "authenticate", Data: map[string]string{"secret": "wrong"}}
	sendAndCheck(t, auth)
	var response ErrorEvent
	err := websocket.JSON.Receive(ws, &response)
	if err != nil {
		t.Error(err)
	}
	if response.Error != "not_authenticated" {
		t.Errorf("Expected `not_authenticated` error, `%s` given", response.Error)
	}
	auth = DataEvent{Event: "authenticate", Data: map[string]string{"secret": "secret"}}
	sendAndCheck(t, auth)
	checkOkResponse(t)
}

func TestEvents(t *testing.T) {
	event := DataEvent{Event: "my-custom-event", Channel: "wrong", Data: map[string]string{"foo":"bar"}}
	sendAndCheck(t, event)
	var response ErrorEvent
	err := websocket.JSON.Receive(ws, &response)
	if err != nil {
		t.Error(err)
	}
	if response.Error != "invalid_channel" {
		t.Errorf("Expected `invalid_channel` error, `%s` given", response.Error)
	}
	event.Channel = "test"
	ws2, err := websocket.Dial("ws://localhost:9771/echo", "ws", "http://localhost/")
	if err != nil {
		t.Error(err)
	}
	subscr := ChanneledEvent{Event: "subscribe", Channel: "test"}
	err = websocket.JSON.Send(ws2, subscr)
	if err != nil {
		t.Error(err)
	}
	var response2 NamedEvent
	err = websocket.JSON.Receive(ws2, &response2)
	if response2.Event != "ok" {
		t.Errorf("Expected `ok` event, `%s` given", response2.Event)
	}
	sendAndCheck(t, event)
	checkOkResponse(t)
	var recvEvent DataEvent
	err = websocket.JSON.Receive(ws2, &recvEvent)
	if err != nil {
		t.Error(err)
	}
	if recvEvent.Event != "my-custom-event" || recvEvent.Data["foo"] != "bar" {
		t.Errorf("Received event differs from original one (%s) != (%s)", recvEvent, event)
	}
	err = websocket.JSON.Send(ws2, event)
	if err != nil {
		t.Error(err)
	}
	err = websocket.JSON.Receive(ws2, &response)
	if err != nil {
		t.Error(err)
	}
	if response.Error != "access_denied" {
		t.Errorf("Expected `access_denied` error, `%s` given", response.Error)
	}
}
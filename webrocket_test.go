package webrocket

import (
	"websocket"
	"testing"
	"os"
	"log"
	"bytes"
)

var (
	ws      *websocket.Conn
	err     os.Error
	secrets *Credentials
)

func init() {
	go func() {
		logger := log.New(bytes.NewBuffer([]byte{}), "", log.LstdFlags)
		server := NewServer(":9771")
		server.Log = logger
		handler := NewHandler(websocket.JSON)
		//handler.Log = logger
		handler.Secrets = Credentials{"read-secret", "read-write-secret"}
		secrets = &handler.Secrets
		server.Handle("/echo", handler)
		server.ListenAndServe()
	}()
}

func wsSendJSON(t *testing.T, data interface{}) {
	err = websocket.JSON.Send(ws, data)
	if err != nil {
		t.Error(err)
	}
}

func wsReadResponse(t *testing.T) Payload {
	var resp Payload
	err := websocket.JSON.Receive(ws, &resp)
	if err != nil {
		t.Error(err)
	}
	return resp
}

func TestConnect(t *testing.T) {
	ws, err = websocket.Dial("ws://localhost:9771/echo", "ws", "http://localhost/")
	if err != nil {
		t.Error(err)
	}
}

func TestAuthInvalidCredentials(t *testing.T) {
	data := Payload{
		"authenticate": Data{
			"access": "read-write",
			"secret": "invalid-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CREDENTIALS" {
		t.Error("Expected invalid credentials error response, given: %s", resp)
	}
}

func TestAuthInvalidData(t *testing.T) {
	data := Payload{
		"authenticate": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestAuthWithMissingAccessType(t *testing.T) {
	data := Payload{
		"authenticate": Data{"secret": "foo"},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestAuthWithMissingSecret(t *testing.T) {
	data := Payload{
		"authenticate": Data{"access": "read-write"},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestAuthWithInvalidAccessTypeType(t *testing.T) {
	data := Payload{
		"authenticate": Data{"access": "invalid", "secret": ""},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestAuthInvalidAccessTypeValue(t *testing.T) {
	data := Payload{
		"authenticate": Data{"access": Data{"foo": "bar"}, "secret": ""},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestAuthAsSubscriber(t *testing.T) {
	data := Payload{
		"authenticate": Data{
			"access": "read-only",
			"secret": "read-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["authenticated"] != "read-only" {
		t.Error("Expected OK response, given: %s", resp)
	}
}

func TestAuthAsPublisher(t *testing.T) {
	data := Payload{
		"authenticate": Data{
			"access": "read-write",
			"secret": "read-write-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["authenticated"] != "read-write" {
		t.Error("Expected success response, given: %s", resp)
	}
}

func TestAuthWithNoSecret(t *testing.T) {
	secrets.ReadWrite = ""
	data := Payload{
		"authenticate": Data{
			"access": "read-write",
			"secret": "",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["authenticated"] != "read-write" {
		t.Error("Expected OK response, given: %s", resp)
	}
	secrets.ReadWrite = "read-write-secret"
}

func TestSubscribeWithInvalidChannel(t *testing.T) {
	data := Payload{
		"subscribe": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestSubscribeWithMissingChannelName(t *testing.T) {
	data := Payload{
		"subscribe": Data{"foo": "bar"},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestSubscribeWithEmptyChannelName(t *testing.T) {
	data := Payload{
		"subscribe": "",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestSubscribeWithInvalidChannelName(t *testing.T) {
	data := Payload{
		"subscribe": Data{"channel": Data{"foo": "bar"}},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestSubscribeWithoutReadAccess(t *testing.T) {
	TestAuthInvalidCredentials(t)
	data := Payload{
		"subscribe": Data{
			"channel": "hello",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "ACCESS_DENIED" {
		t.Error("Expected access denied error response, given: %s", resp)
	}
}

func TestSubscribeWithReadAccess(t *testing.T) {
	TestAuthAsSubscriber(t)
	data := Payload{
		"subscribe": Data{
			"channel": "hello",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["subscribed"] != "hello" {
		t.Error("Expected success response, given: %s", resp)
	}
}

func TestSubscribeWithReadWriteAccess(t *testing.T) {
	TestAuthAsPublisher(t)
	data := Payload{
		"subscribe": Data{
			"channel": "hello",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["subscribed"] != "hello" {
		t.Error("Expected success response, given: %s", resp)
	}
}

func TestUnsubscribeWithInvalidData(t *testing.T) {
	data := Payload{
		"unsubscribe": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestUnsubscribeWithEmptyChannelName(t *testing.T) {
	data := Payload{
		"unsubscribe": Data{"channel": ""},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestUnsubscribeWithInvalidChannelName(t *testing.T) {
	data := Payload{
		"unsubscribe": Data{"channel": Data{"foo": "bar"}},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestUnsubscribeWithMissingChannelName(t *testing.T) {
	data := Payload{
		"unsubscribe": Data{"foo": "bar"},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestUnsubscribe(t *testing.T) {
	data := Payload{
		"unsubscribe": Data{
			"channel": "hello",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["unsubscribed"] != "hello" {
		t.Error("Expected success response, given: %s", resp)
	}
}

func TestBroadcastAsSubscriber(t *testing.T) {
	TestAuthAsSubscriber(t)
	data := Payload{
		"broadcast": Data{
			"channel": "hello",
			"event":   "foo",
			"data":    "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "ACCESS_DENIED" {
		t.Error("Expected access denied error response, given: %s", resp)
	}
}

func TestBroadcastInvalidData(t *testing.T) {
	TestAuthAsPublisher(t)
	data := Payload{
		"broadcast": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestBroadcastInvalidChannel(t *testing.T) {
	data := Payload{
		"broadcast": Data{
			"channel": "invalid-channel",
			"event":   "foo",
			"data":    "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CHANNEL" {
		t.Error("Expected invalid channel error response, given: %s", resp)
	}
}

func TestBroadcastInvalidChannelName(t *testing.T) {
	data := Payload{
		"broadcast": Data{
			"channel": Data{"foo": "bar"},
			"event":   "foo",
			"data":    "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid channel error response, given: %s", resp)
	}
}

func TestBroadcastWithMissingEvent(t *testing.T) {
	data := Payload{
		"broadcast": Data{
			"channel": "hello",
			"data":    "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid data error response, given: %s", resp)
	}
}

func TestBroadcastMissingChannel(t *testing.T) {
	data := Payload{
		"broadcast": Data{
			"event":   "foo",
			"data":    "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Error("Expected invalid channel error response, given: %s", resp)
	}
}

func TestBroadcast(t *testing.T) {
	var resp Payload
	ws2, _ := websocket.Dial("ws://localhost:9771/echo", "ws", "http://localhost/")
	websocket.JSON.Send(ws2, Payload{
		"authenticate": Data{
			"access": "read-only",
			"secret": "read-secret",
		},
	})
	websocket.JSON.Receive(ws2, resp)
	websocket.JSON.Send(ws2, Payload{
		"subscribe": Data{
			"channel": "hello",
		},
	})
	websocket.JSON.Receive(ws2, resp)
	TestAuthAsPublisher(t)
	wsSendJSON(t, Payload{
		"broadcast": Data{
			"channel": "hello",
			"event":   "foo",
			"data":    "bar",
		},
	})
	resp = wsReadResponse(t)
	if resp["broadcasted"] != "hello" {
		t.Error("Expected success response, given: %s", resp)
	}
	err = websocket.JSON.Receive(ws2, &resp)
	if err != nil {
		t.Error(err)
	}
	if resp["event"] != "foo" && resp["data"] != "bar" {
		t.Error("Invalid broadcast: %s", resp)
	}
}

func TestLogout(t *testing.T) {
	data := Payload{"logout": true}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["loggedOut"] != true {
		t.Error("Expected success  response, given: %s", resp)
	}
}

func TestDisconnect(t *testing.T) {
	data := Payload{"disconnect": true}
	wsSendJSON(t, data)
	_, err := ws.Read(make([]uint8, 1))
	if err != os.EOF {
		t.Error("Expected EOF, given: %s", err)
	}
}

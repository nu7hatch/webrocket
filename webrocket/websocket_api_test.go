// This package provides a hybrid of MQ and WebSockets server with
// support for horizontal scalability.
//
// Copyright (C) 2011 by Krzysztof Kowalik <chris@nu7hat.ch>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.
package webrocket

import (
	"bytes"
	"io"
	"log"
	"testing"
	"websocket"
)

var (
	ws  *websocket.Conn
	err error
)

func init() {
	go func() {
		server := NewServer(":9771")
		server.Log = log.New(bytes.NewBuffer([]byte{}), "a", log.LstdFlags)
		vhost, _ := server.AddVhost("/echo")
		vhost.AddUser("front", "read-secret", PermRead)
		vhost.AddUser("back", "read-write-secret", PermRead|PermWrite)
		vhost.AddUser("no-secret", "", PermRead)
		server.ListenAndServe()
	}()
}

func wsSendJSON(t *testing.T, data interface{}) {
	err = websocket.JSON.Send(ws, data)
	if err != nil {
		t.Error(err)
	}
}

func wsReadResponse(t *testing.T) map[string]interface{} {
	var resp map[string]interface{}
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
	data := map[string]interface{}{
		"authenticate": map[string]string{
			"user":   "front",
			"secret": "invalid-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CREDENTIALS" {
		t.Errorf("Expected invalid credentials error response, given: %s", resp)
	}
}

func TestAuthInvalidData(t *testing.T) {
	data := map[string]interface{}{
		"authenticate": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Errorf("Expected invalid payload error response, given: %s", resp)
	}
}

func TestAuthWithMissingUserName(t *testing.T) {
	data := map[string]interface{}{
		"authenticate": map[string]string{
			"secret": "foo",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_USER_NAME" {
		t.Errorf("Expected invalid user name error response, given: %s", resp)
	}
}

func TestAuthWithMissingSecret(t *testing.T) {
	data := map[string]interface{}{
		"authenticate": map[string]string{
			"user": "front",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CREDENTIALS" {
		t.Errorf("Expected invalid credentials error response, given: %s", resp)
	}
}

func TestAuthWithInvalidUser(t *testing.T) {
	data := map[string]interface{}{
		"authenticate": map[string]string{
			"user": "invalid",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "USER_NOT_FOUND" {
		t.Errorf("Expected user not found error response, given: %s", resp)
	}
}

func TestAuthInvalidUserValue(t *testing.T) {
	data := map[string]interface{}{
		"authenticate": map[string]interface{}{
			"user":   map[string]string{"foo": "bar"},
			"secret": "",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_USER_NAME" {
		t.Errorf("Expected invalid user name error response, given: %s", resp)
	}
}

func TestAuthInvalidSecretValue(t *testing.T) {
	data := map[string]interface{}{
		"authenticate": map[string]interface{}{
			"user":   "front",
			"secret": map[string]string{"foo": "bar"},
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CREDENTIALS" {
		t.Errorf("Expected invalid credentials error response, given: %s", resp)
	}
}

func TestAuthAsSubscriber(t *testing.T) {
	data := map[string]interface{}{
		"authenticate": map[string]string{
			"user":   "front",
			"secret": "read-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["authenticated"] != "front" {
		t.Errorf("Expected authenticated response, given: %s", resp)
	}
}

func TestAuthAsPublisher(t *testing.T) {
	data := map[string]interface{}{
		"authenticate": map[string]string{
			"user":   "back",
			"secret": "read-write-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["authenticated"] != "back" {
		t.Errorf("Expected authenticated response, given: %s", resp)
	}
}

func TestAuthWithNoSecret(t *testing.T) {
	data := map[string]interface{}{
		"authenticate": map[string]string{
			"user": "no-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["authenticated"] != "no-secret" {
		t.Errorf("Expected authenticated response, given: %s", resp)
	}
}

func TestSubscribeWithInvalidData(t *testing.T) {
	data := map[string]string{
		"subscribe": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Errorf("Expected invalid payload error response, given: %s", resp)
	}
}

func TestSubscribeWithMissingChannelName(t *testing.T) {
	data := map[string]interface{}{
		"subscribe": map[string]string{"foo": "bar"},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CHANNEL_NAME" {
		t.Errorf("Expected invalid channel name error response, given: %s", resp)
	}
}

func TestSubscribeWithEmptyChannelName(t *testing.T) {
	data := map[string]interface{}{
		"subscribe": map[string]string{
			"channel": "",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CHANNEL_NAME" {
		t.Errorf("Expected invalid channel name error response, given: %s", resp)
	}
}

func TestSubscribeWithInvalidChannelName(t *testing.T) {
	data := map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": map[string]string{"foo": "bar"},
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CHANNEL_NAME" {
		t.Errorf("Expected invalid channel name error response, given: %s", resp)
	}
}

func TestSubscribeWithNoAccess(t *testing.T) {
	TestLogout(t)
	data := map[string]interface{}{
		"subscribe": map[string]string{
			"channel": "hello",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "ACCESS_DENIED" {
		t.Errorf("Expected access denied error response, given: %s", resp)
	}
}

func TestSubscribeWithReadWriteAccess(t *testing.T) {
	TestAuthAsSubscriber(t)
	data := map[string]interface{}{
		"subscribe": map[string]string{
			"channel": "hello",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["subscribed"] != "hello" {
		t.Errorf("Expected subscribed response, given: %s", resp)
	}
}

func TestUnsubscribeWithInvalidData(t *testing.T) {
	data := map[string]string{
		"unsubscribe": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Errorf("Expected invalid payload error response, given: %s", resp)
	}
}

func TestUnsubscribeWithEmptyChannelName(t *testing.T) {
	data := map[string]interface{}{
		"unsubscribe": map[string]string{
			"channel": "",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CHANNEL_NAME" {
		t.Errorf("Expected invalid channel name error response, given: %s", resp)
	}
}

func TestUnsubscribeWithInvalidChannelName(t *testing.T) {
	data := map[string]interface{}{
		"unsubscribe": map[string]interface{}{
			"channel": map[string]string{
				"foo": "bar",
			},
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CHANNEL_NAME" {
		t.Errorf("Expected invalid channel name error response, given: %s", resp)
	}
}

func TestUnsubscribeWithMissingChannelName(t *testing.T) {
	data := map[string]interface{}{
		"unsubscribe": map[string]string{
			"foo": "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CHANNEL_NAME" {
		t.Errorf("Expected invalid channel name error response, given: %s", resp)
	}
}

func TestUnsubscribe(t *testing.T) {
	data := map[string]interface{}{
		"unsubscribe": map[string]string{
			"channel": "hello",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["unsubscribed"] != "hello" {
		t.Errorf("Expected unsubscribed response, given: %s", resp)
	}
}

func TestBroadcastAsSubscriber(t *testing.T) {
	TestAuthAsSubscriber(t)
	data := map[string]interface{}{
		"broadcast": map[string]string{
			"channel": "hello",
			"event":   "foo",
			"data":    "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "ACCESS_DENIED" {
		t.Errorf("Expected access denied error response, given: %s", resp)
	}
}

func TestBroadcastInvalidData(t *testing.T) {
	TestAuthAsPublisher(t)
	data := map[string]string{
		"broadcast": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_PAYLOAD" {
		t.Errorf("Expected invalid payload error response, given: %s", resp)
	}
}

func TestBroadcastInvalidChannel(t *testing.T) {
	data := map[string]interface{}{
		"broadcast": map[string]string{
			"channel": "invalid-channel",
			"event":   "foo",
			"data":    "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "CHANNEL_NOT_FOUND" {
		t.Errorf("Expected channel not found error response, given: %s", resp)
	}
}

func TestBroadcastInvalidChannelName(t *testing.T) {
	data := map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": map[string]string{"foo": "bar"},
			"event":   "foo",
			"data":    "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CHANNEL_NAME" {
		t.Errorf("Expected invalid channel name error response, given: %s", resp)
	}
}

func TestBroadcastWithMissingEvent(t *testing.T) {
	data := map[string]interface{}{
		"broadcast": map[string]string{
			"channel": "hello",
			"data":    "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_EVENT_NAME" {
		t.Errorf("Expected invalid event name error response, given: %s", resp)
	}
}

func TestBroadcastMissingChannel(t *testing.T) {
	data := map[string]interface{}{
		"broadcast": map[string]string{
			"event": "foo",
			"data":  "bar",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["err"] != "INVALID_CHANNEL_NAME" {
		t.Errorf("Expected invalid channel name error response, given: %s", resp)
	}
}

func TestBroadcast(t *testing.T) {
	var resp map[string]interface{}
	ws2, _ := websocket.Dial("ws://localhost:9771/echo", "ws", "http://localhost/")
	websocket.JSON.Send(ws2, map[string]interface{}{
		"authenticate": map[string]string{
			"user":   "front",
			"secret": "read-secret",
		},
	})
	websocket.JSON.Receive(ws2, resp)
	websocket.JSON.Send(ws2, map[string]interface{}{
		"subscribe": map[string]string{
			"channel": "hello",
		},
	})
	websocket.JSON.Receive(ws2, resp)
	TestAuthAsPublisher(t)
	wsSendJSON(t, map[string]interface{}{
		"broadcast": map[string]string{
			"channel": "hello",
			"event":   "foo",
			"data":    "bar",
		},
	})
	resp = wsReadResponse(t)
	if resp["broadcasted"] != "hello" {
		t.Errorf("Expected broadcasted response, given: %s", resp)
	}
	err = websocket.JSON.Receive(ws2, &resp)
	if err != nil {
		t.Errorf(err.Error())
	}
	if resp["event"] != "foo" && resp["data"] != "bar" {
		t.Errorf("Invalid broadcast: %s", resp)
	}
}

func TestLogout(t *testing.T) {
	data := map[string]bool{"logout": true}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if resp["loggedOut"] != true {
		t.Errorf("Expected logged out response, given: %s", resp)
	}
}

func TestDisconnect(t *testing.T) {
	data := map[string]bool{"disconnect": true}
	wsSendJSON(t, data)
	_, err := ws.Read(make([]uint8, 1))
	if err != io.EOF {
		t.Errorf("Expected EOF, given: %s", err)
	}
}

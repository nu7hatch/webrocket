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
	ws     *websocket.Conn
	err    error
	server *WebsocketServer
	vhost  *Vhost
)

func init() {
	go func() {
		ctx := NewContext()
		ctx.Log = log.New(bytes.NewBuffer([]byte{}), "", log.LstdFlags)
		server = ctx.NewWebsocketServer(":9771")
		vhost, _ = ctx.AddVhost("/echo")
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

func extractErr(data map[string]interface{}) string {
	d, ok := data["__error"].(map[string]interface{})
	if !ok {
		return ""
	}
	return d["id"].(string)
}

func TestConnect(t *testing.T) {
	ws, err = websocket.Dial("ws://localhost:9771/echo", "ws", "http://localhost/")
	if err != nil {
		t.Error(err)
	}
}

func TestInvalidDataReceived(t *testing.T) {
	_, err = ws.Write([]byte("foobar"))
	if err != nil {
		t.Error(err)
	}
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_DATA_RECEIVED" {
		t.Errorf("Expected invalid data received error response, given: %s", resp)
	}
}

func TestInvalidMessageFormat(t *testing.T) {
	_, err = ws.Write([]byte("{}"))
	if err != nil {
		t.Error(err)
	}
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_MESSAGE_FORMAT" {
		t.Errorf("Expected invalid message format error response, given: %s", resp)
	}
}

func TestAuthInvalidCredentials(t *testing.T) {
	data := map[string]interface{}{
		"auth": map[string]string{
			"user":   "front",
			"secret": "invalid-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_CREDENTIALS" {
		t.Errorf("Expected invalid credentials error response, given: %s", resp)
	}
}

func TestAuthInvalidData(t *testing.T) {
	data := map[string]interface{}{
		"auth": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_PAYLOAD" {
		t.Errorf("Expected invalid payload error response, given: %s", resp)
	}
}

func TestAuthWithMissingUserName(t *testing.T) {
	data := map[string]interface{}{
		"auth": map[string]string{
			"secret": "foo",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_USER_NAME" {
		t.Errorf("Expected invalid user name error response, given: %s", resp)
	}
}

func TestAuthWithMissingSecret(t *testing.T) {
	data := map[string]interface{}{
		"auth": map[string]string{
			"user": "front",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_CREDENTIALS" {
		t.Errorf("Expected invalid credentials error response, given: %s", resp)
	}
}

func TestAuthWithInvalidUser(t *testing.T) {
	data := map[string]interface{}{
		"auth": map[string]string{
			"user": "invalid",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "USER_NOT_FOUND" {
		t.Errorf("Expected user not found error response, given: %s", resp)
	}
}

func TestAuthInvalidUserValue(t *testing.T) {
	data := map[string]interface{}{
		"auth": map[string]interface{}{
			"user":   map[string]string{"foo": "bar"},
			"secret": "",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_USER_NAME" {
		t.Errorf("Expected invalid user name error response, given: %s", resp)
	}
}

func TestAuthInvalidSecretValue(t *testing.T) {
	data := map[string]interface{}{
		"auth": map[string]interface{}{
			"user":   "front",
			"secret": map[string]string{"foo": "bar"},
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_CREDENTIALS" {
		t.Errorf("Expected invalid credentials error response, given: %s", resp)
	}
}

func TestAuthAsSubscriber(t *testing.T) {
	data := map[string]interface{}{
		"auth": map[string]string{
			"user":   "front",
			"secret": "read-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	d, ok := resp["__authenticated"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected authenticated response, given: %s", resp)
	}
	user, _ := d["user"].(string)
	if user != "front" {
		t.Errorf("Expected to auth as 'front', given: %s", user)
	}
}

func TestAuthAsPublisher(t *testing.T) {
	data := map[string]interface{}{
		"auth": map[string]string{
			"user":   "back",
			"secret": "read-write-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	d, ok := resp["__authenticated"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected authenticated response, given: %s", resp)
	}
	user, _ := d["user"].(string)
	if user != "back" {
		t.Errorf("Expected to auth as 'back', given: %s", user)
	}
}

func TestAuthWithNoSecret(t *testing.T) {
	data := map[string]interface{}{
		"auth": map[string]string{
			"user": "no-secret",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	d, ok := resp["__authenticated"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected authenticated response, given: %s", resp)
	}
	user, _ := d["user"].(string)
	if user != "no-secret" {
		t.Errorf("Expected to auth as 'no-secret', given: %s", user)
	}
}

func TestSubscribeWithInvalidData(t *testing.T) {
	data := map[string]string{
		"subscribe": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_PAYLOAD" {
		t.Errorf("Expected invalid payload error response, given: %s", resp)
	}
}

func TestSubscribeWithMissingChannelName(t *testing.T) {
	data := map[string]interface{}{
		"subscribe": map[string]string{"foo": "bar"},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_CHANNEL_NAME" {
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
	if extractErr(resp) != "INVALID_CHANNEL_NAME" {
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
	if extractErr(resp) != "INVALID_CHANNEL_NAME" {
		t.Errorf("Expected invalid channel name error response, given: %s", resp)
	}
}

func TestSubscribeWithNoAccess(t *testing.T) {
	TestAuthInvalidCredentials(t)
	data := map[string]interface{}{
		"subscribe": map[string]string{
			"channel": "hello",
		},
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "ACCESS_DENIED" {
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
	d, ok := resp["__subscribed"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected subscribed response, given: %s", resp)
	}
	channel, _ := d["channel"].(string)
	if channel != "hello" {
		t.Errorf("Expected to subscribe hello, given: %s", channel)
	}
}

func TestUnsubscribeWithInvalidData(t *testing.T) {
	data := map[string]string{
		"unsubscribe": "invalid",
	}
	wsSendJSON(t, data)
	resp := wsReadResponse(t)
	if extractErr(resp) != "INVALID_PAYLOAD" {
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
	if extractErr(resp) != "INVALID_CHANNEL_NAME" {
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
	if extractErr(resp) != "INVALID_CHANNEL_NAME" {
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
	if extractErr(resp) != "INVALID_CHANNEL_NAME" {
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
	hello, _ := vhost.GetChannel("hello")
	for _, s := range hello.Subscribers() {
		if s.Conn == ws {
			t.Errorf("Expected to unsubscribe the 'hello' channel")
		}
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
	if extractErr(resp) != "ACCESS_DENIED" {
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
	if extractErr(resp) != "INVALID_PAYLOAD" {
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
	if extractErr(resp) != "CHANNEL_NOT_FOUND" {
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
	if extractErr(resp) != "INVALID_CHANNEL_NAME" {
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
	if extractErr(resp) != "INVALID_EVENT_NAME" {
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
	if extractErr(resp) != "INVALID_CHANNEL_NAME" {
		t.Errorf("Expected invalid channel name error response, given: %s", resp)
	}
}

func TestBroadcast(t *testing.T) {
	var resp map[string]interface{}
	ws2, _ := websocket.Dial("ws://localhost:9771/echo", "ws", "http://localhost/")
	websocket.JSON.Send(ws2, map[string]interface{}{
		"auth": map[string]string{
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
	err = websocket.JSON.Receive(ws2, &resp)
	if err != nil {
		t.Errorf(err.Error())
	}
	if resp["event"] != "foo" && resp["data"] != "bar" {
		t.Errorf("Invalid broadcast: %s", resp)
	}
}

func TestClose(t *testing.T) {
	data := map[string]bool{"close": true}
	wsSendJSON(t, data)
	_, err := ws.Read(make([]uint8, 1))
	if err != io.EOF {
		t.Errorf("Expected EOF, given: %s", err)
	}
}

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
	"fmt"
	"log"
	"testing"
	"time"
	"websocket"
)

type testingT interface {
	Errorf(format string, args ...interface{})
	Error(args ...interface{})
}

var (
	ws   *websocket.Conn
	werr error
	we   Endpoint
	wv   *Vhost
)

func init() {
	ctx := NewContext()
	ctx.SetLog(log.New(bytes.NewBuffer([]byte{}), "", log.LstdFlags))
	we = ctx.NewWebsocketEndpoint("", 9771)
	wv, _ = ctx.AddVhost("/test")
	wv.OpenChannel("test")
	wv.OpenChannel("test2")
	go we.ListenAndServe()
}

func wssend(t testingT, data interface{}) {
	wssendto(t, ws, data)
}

func wssendto(t testingT, ws *websocket.Conn, data interface{}) {
	werr = websocket.JSON.Send(ws, data)
	if werr != nil {
		t.Error(werr)
	}
}

func wsrecv(t testingT) *Message {
	return wsrecvfrom(t, ws)
}

func wsrecvfrom(t testingT, ws *websocket.Conn) *Message {
	var resp map[string]interface{}
	werr := websocket.JSON.Receive(ws, &resp)
	if werr != nil {
		t.Error(werr)
		return nil
	}
	msg, werr := newMessage(resp)
	if werr != nil {
		t.Error(werr)
	}
	return msg
}

func wserr(msg *Message) string {
	s, _ := msg.Get("status").(string)
	return s
}

func wsdial(port int) (*websocket.Conn, error) {
	url := fmt.Sprintf("ws://127.0.0.1:%d/test", port)
	return websocket.Dial(url, "ws", "http://127.0.0.1/")
}

func doTestWebsocketConnect(t *testing.T) {
	ws, werr = wsdial(9771)
	if werr != nil {
		t.Error(werr)
	}
	resp := wsrecv(t)
	if resp.Event() != "__connected" {
		t.Errorf("Expected to receive the '__connected' event, given '%s'", resp.Event())
	}
}

func doTestWebsocketBadRequest(t *testing.T) {
	ws.Write([]byte("foobar"))
	resp := wsrecv(t)
	if wserr(resp) != "Bad request" {
		t.Errorf("Expected 'Bad request' error")
	}
	ws.Write([]byte("{}"))
	resp = wsrecv(t)
	if wserr(resp) != "Bad request" {
		t.Errorf("Expected 'Bad request' error")
	}
}

func doTestWebsocketNotFound(t *testing.T) {
	ws.Write([]byte("{\"hello\": {}}"))
	resp := wsrecv(t)
	if wserr(resp) != "Bad request" {
		t.Errorf("Expected 'Bad request' error")
	}
}

func doTestWebsocketAuthWithMissingToken(t *testing.T) {
	wssend(t, map[string]interface{}{
		"auth": map[string]interface{}{"foo": "bar"},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Bad request" {
		t.Errorf("Expected 'Bad request' error")
	}
}

func doTestWebsocketAuthWuthInvalidTokenValue(t *testing.T) {
	wssend(t, map[string]interface{}{
		"auth": map[string]interface{}{"tokena": map[string]interface{}{}},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Bad request" {
		t.Errorf("Expected 'Bad request' error")
	}
}

func doTestWebsocketAuthWithInvalidToken(t *testing.T) {
	wssend(t, map[string]interface{}{
		"auth": map[string]interface{}{"token": "invalid"},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Unauthorized" {
		t.Errorf("Expected 'Unauthorized' error")
	}
}

func doTestWebsocketAuthWithValidToken(t *testing.T) {
	token := wv.GenerateSingleAccessToken(".*")
	wssend(t, map[string]interface{}{
		"auth": map[string]interface{}{
			"token": token,
		},
	})
	resp := wsrecv(t)
	if resp.Event() != "__authenticated" {
		t.Errorf("Expected to receive the '__authenticated' event, given '%s'", resp.Event())
	}
}

func doTestWebsocketSubscribeInvalidChannelName(t *testing.T) {
	wssend(t, map[string]interface{}{
		"subscribe": map[string]interface{}{"channel": "shit%dfsdf%#"},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Channel not found" {
		t.Errorf("Expected 'Channel not found' error")
	}
}

func doTestWebsocketSubscribeEmptyChannelName(t *testing.T) {
	wssend(t, map[string]interface{}{
		"subscribe": map[string]interface{}{"channel": ""},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Bad request" {
		t.Errorf("Expected 'Bad request' error")
	}
}

func doTestWebsocketSubscribeNoChannelName(t *testing.T) {
	wssend(t, map[string]interface{}{
		"subscribe": map[string]interface{}{"foo": ""},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Bad request" {
		t.Errorf("Expected 'Bad request' error")
	}
}

func doTestWebsocketSubscribeAllowedChannel(t *testing.T) {
	wssend(t, map[string]interface{}{
		"subscribe": map[string]interface{}{"channel": "test"},
	})
	resp := wsrecv(t)
	if resp.Event() != "__subscribed" {
		t.Errorf("Expected to receive the '__subscribed' event, given '%s'", resp.Event())
	}
}

func doTestWebsocketUnsubscribeEmptyChannelName(t *testing.T) {
	wssend(t, map[string]interface{}{
		"unsubscribe": map[string]interface{}{"channel": ""},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Bad request" {
		t.Errorf("Expected 'Bad request' error")
	}
}

func doTestWebsocketUnsubscribeInvalidChannelName(t *testing.T) {
	wssend(t, map[string]interface{}{
		"unsubscribe": map[string]interface{}{"channel": "shit%dfsdf%#"},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Channel not found" {
		t.Errorf("Expected 'Channel not found' error")
	}
}

func doTestWebsocketUnsubscribeNotSubscribedChannel(t *testing.T) {
	wssend(t, map[string]interface{}{
		"unsubscribe": map[string]interface{}{"channel": "test2"},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Not subscribed" {
		t.Errorf("Expected 'Not subscribed' error")
	}
}

func doTestWebsocketUnsubscribeValidChannel(t *testing.T) {
	wssend(t, map[string]interface{}{
		"unsubscribe": map[string]interface{}{"channel": "test"},
	})
	resp := wsrecv(t)
	if resp.Event() != "__unsubscribed" {
		t.Errorf("Expected to receive the '__unsubscribed' event, given '%s'", resp.Event())
	}
}

func doTestWebsocketBroadcastWhenNotSubscribingTheChannel(t *testing.T) {
	wssend(t, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "test", "event": "foo", "data": map[string]interface{}{},
		},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Not subscribed" {
		t.Errorf("Expected 'Not subscibed' error")
	}
}

func doTestWebsocketBroadcastWithInvalidData(t *testing.T) {
	wssend(t, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "test", "data": map[string]interface{}{},
		},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Bad request" {
		t.Errorf("Expected 'Bad request' error")
	}
	wssend(t, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"event": "test", "data": map[string]interface{}{},
		},
	})
	resp = wsrecv(t)
	if wserr(resp) != "Bad request" {
		t.Errorf("Expected 'Bad request' error")
	}
}

func doTestWebsocketBroadcastWhenInvalidChannelGiven(t *testing.T) {
	wssend(t, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "notexists", "event": "foo", "data": map[string]interface{}{},
		},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Channel not found" {
		t.Errorf("Expected 'Channel not found' error")
	}
}

func doTestWebsocketBroadcastToNotSubscribedChannel(t *testing.T) {
	wssend(t, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "test2", "event": "foo", "data": map[string]interface{}{},
		},
	})
	resp := wsrecv(t)
	if wserr(resp) != "Not subscribed" {
		t.Errorf("Expected 'Not subscribed' error")
	}
}

func doTestWebsocketBroadcastValidData(t *testing.T) {
	var wss [2]*websocket.Conn
	for i := range wss {
		wss[i], _ = wsdial(9771)
		wsrecvfrom(t, wss[i])
		wssendto(t, wss[i], map[string]interface{}{
			"subscribe": map[string]interface{}{"channel": "test"},
		})
		wsrecvfrom(t, wss[i])
	}
	wssend(t, map[string]interface{}{
		"subscribe": map[string]interface{}{"channel": "test"},
	})
	wsrecv(t)
	wssend(t, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "test",
			"event":   "hello",
			"data":    map[string]interface{}{"foo": "bar"},
		},
	})
	time.Sleep(1e6)
	for i := range wss {
		resp := wsrecvfrom(t, wss[i])
		if resp.Event() != "hello" {
			t.Errorf("Expected to broadcast the 'hello' event")
		}
		chanName, _ := resp.Get("channel").(string)
		if chanName != "test" {
			t.Errorf("Expected to broadcast on the 'test' channel")
		}
		sid, _ := resp.Get("sid").(string)
		if sid == "" {
			t.Errorf("Expected to append sender's sid to the broadcasted data")
		}
		foo, _ := resp.Get("foo").(string)
		if foo != "bar" {
			t.Errorf("Expected to broadcast passed data")
		}
	}
}

// TODO: test 'broadcast' with tigger
// TODO: test 'subscribe' and 'broadcast' for protected channels
// TODO: test 'trigger'
// TODO: test 'close'

func TestWebsocketProtocol(t *testing.T) {
	// It kind of sucks, but it's an integration test where each
	// steps may depend on the others, so we have to run it in
	// correct order.
	// FIXME: Find some way to do it more nicely...
	doTestWebsocketConnect(t)
	doTestWebsocketBadRequest(t)
	doTestWebsocketNotFound(t)
	doTestWebsocketAuthWithMissingToken(t)
	doTestWebsocketAuthWuthInvalidTokenValue(t)
	doTestWebsocketAuthWithInvalidToken(t)
	doTestWebsocketAuthWithValidToken(t)
	doTestWebsocketSubscribeInvalidChannelName(t)
	doTestWebsocketSubscribeEmptyChannelName(t)
	doTestWebsocketSubscribeNoChannelName(t)
	doTestWebsocketSubscribeAllowedChannel(t)
	doTestWebsocketUnsubscribeEmptyChannelName(t)
	doTestWebsocketUnsubscribeInvalidChannelName(t)
	doTestWebsocketUnsubscribeNotSubscribedChannel(t)
	doTestWebsocketUnsubscribeValidChannel(t)
	doTestWebsocketBroadcastWhenNotSubscribingTheChannel(t)
	doTestWebsocketBroadcastWithInvalidData(t)
	doTestWebsocketBroadcastWhenInvalidChannelGiven(t)
	doTestWebsocketBroadcastToNotSubscribedChannel(t)
	doTestWebsocketBroadcastValidData(t)
}
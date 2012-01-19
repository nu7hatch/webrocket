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
	"os"
	"regexp"
	"testing"
	"websocket"
)

var ctx *Context

func init() {
	ctx = NewContext()
	ctx.NewWebsocketEndpoint(":9080")
	go ctx.websocket.ListenAndServe()
	ctx.NewBackendEndpoint(":9081")
	go ctx.backend.ListenAndServe()
	v, _ := ctx.AddVhost("/test")
	v.OpenChannel("test", ChannelNormal)
	v.OpenChannel("private-test", ChannelPrivate)
	v.OpenChannel("presence-test", ChannelPresence)
}

func websocketExpectResponse(t *testing.T, ws *websocket.Conn, event string, data map[string]*regexp.Regexp) {
	var resp map[string]interface{}
	var msg *WebsocketMessage
	var err error
	if err = websocket.JSON.Receive(ws, &resp); err != nil {
		t.Error(err)
		return
	}
	if msg, err = newWebsocketMessage(resp); err != nil {
		t.Error(err)
		return
	}
	if event != msg.Event() {
		t.Errorf("Expected event to be '%s', got '%s'", event, msg.Event())
	}
	for key, re := range data {
		if value, ok := msg.Get(key).(string); !ok || !re.MatchString(value) {
			t.Errorf("Expected data to contain the proper '%s' value, given '%s'", key, value)
		}
	}
}

func websocketExpectError(t *testing.T, ws *websocket.Conn, status string) {
	websocketExpectResponse(t, ws, "__error", map[string]*regexp.Regexp{
		"status": regexp.MustCompile("^" + status + "$"),
	})
}

func websocketDial(t *testing.T) *websocket.Conn {
	ws, err := websocket.Dial("ws://127.0.0.1:9080/test", "ws", "http://127.0.0.1/")
	if err != nil {
		t.Error(err)
		os.Exit(1)
	}
	return ws
}

func websocketSend(t *testing.T, ws *websocket.Conn, data interface{}) {
	if err := websocket.JSON.Send(ws, data); err != nil {
		t.Error(err)
	}
}

func testWebsocketConnect(t *testing.T, ws *websocket.Conn) {
	websocketExpectResponse(t, ws, "__connected", map[string]*regexp.Regexp{
		"sid": regexp.MustCompile("^.{36}$"),
	})
}

func testWebsocketBadRequests(t *testing.T, ws *websocket.Conn) {
	for _, data := range []string{"foobar", "{}", "{\"hello\": {}}"} {
		websocketSend(t, ws, data)
		websocketExpectError(t, ws, "Bad request")
	}
}

func testWebsocketAuthenticationWithoutToken(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"auth": map[string]interface{}{
			"foo": "bar",
		},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketAuthenticationWithInvalidTokenFormat(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"auth": map[string]interface{}{
			"token": map[string]interface{}{},
		},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketAuthenticationWithInvalidToken(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"auth": map[string]interface{}{
			"token": "invalid",
		},
	})
	websocketExpectError(t, ws, "Unauthorized")
}

func testWebsocketAuthenticationWithValidToken(t *testing.T, ws *websocket.Conn) {
	v, _ := ctx.Vhost("/test")
	token := v.GenerateSingleAccessToken(".*")
	websocketSend(t, ws, map[string]interface{}{
		"auth": map[string]interface{}{
			"token": token,
		},
	})
	websocketExpectResponse(t, ws, "__authenticated", nil)
}

func testWebsocketSubscribeWithoutChannelName(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketSubscribeWithEmptyChannelName(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "",
		},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketSubscribeWithInvalidChannelName(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "#&*^&^&&",
		},
	})
	websocketExpectError(t, ws, "Channel not found")
}

func testWebsocketSubscribePublicChannel(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "test",
		},
	})
	websocketExpectResponse(t, ws, "__subscribed", map[string]*regexp.Regexp{
		"channel": regexp.MustCompile("^test$"),
	})
}

func testWebsocketSubscribePrivateChannelWithoutAuthentication(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "private-test",
		},
	})
	websocketExpectError(t, ws, "Forbidden")
}

func testWebsocketSubscribePresenceChannelWithoutAuthentication(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "presence-test",
		},
	})
	websocketExpectError(t, ws, "Forbidden")
}

func testWebsocketSubscribePrivateChannelWithAuthentication(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "private-test",
		},
	})
	websocketExpectResponse(t, ws, "__subscribed", map[string]*regexp.Regexp{
		"channel": regexp.MustCompile("^private-test$"),
	})
}

func testWebsocketSubscribePresenceChannelWithAuthentication(t *testing.T, ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "presence-test",
		},
	})
	websocketExpectResponse(t, ws, "__subscribed", map[string]*regexp.Regexp{
		"channel": regexp.MustCompile("^presence-test$"),
	})
}

func TestAllTheThings(t *testing.T) {
	ws := websocketDial(t)
	testWebsocketConnect(t, ws)
	testWebsocketBadRequests(t, ws)
	testWebsocketAuthenticationWithoutToken(t, ws)
	testWebsocketAuthenticationWithInvalidTokenFormat(t, ws)
	testWebsocketAuthenticationWithInvalidToken(t, ws)
	testWebsocketAuthenticationWithValidToken(t, ws)
	testWebsocketSubscribeWithoutChannelName(t, ws)
	testWebsocketSubscribeWithEmptyChannelName(t, ws)
	testWebsocketSubscribeWithInvalidChannelName(t, ws)

	ws.Close()
	ws = websocketDial(t)
	testWebsocketConnect(t, ws)
	testWebsocketSubscribePublicChannel(t, ws)
	testWebsocketSubscribePrivateChannelWithoutAuthentication(t, ws)
	testWebsocketSubscribePresenceChannelWithoutAuthentication(t, ws)
	testWebsocketAuthenticationWithValidToken(t, ws)
	testWebsocketSubscribePrivateChannelWithAuthentication(t, ws)
	testWebsocketSubscribePresenceChannelWithAuthentication(t, ws)
}

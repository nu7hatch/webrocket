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
	"net"
	"strings"
	"bufio"
	"fmt"
	"time"
	"../uuid"
)

var (
	ctx *Context
	v *Vhost
)

func init() {
	ctx = NewContext()
	ctx.NewWebsocketEndpoint(":9080")
	go ctx.websocket.ListenAndServe()
	ctx.NewBackendEndpoint(":9081")
	go ctx.backend.ListenAndServe()
	v, _ = ctx.AddVhost("/test")
	v.OpenChannel("test", ChannelNormal)
	v.OpenChannel("private-test", ChannelPrivate)
	v.OpenChannel("presence-test", ChannelPresence)
}

func websocketDial(t *testing.T) *websocket.Conn {
	ws, err := websocket.Dial("ws://127.0.0.1:9080/test", "ws", "http://127.0.0.1/")
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	return ws
}

func websocketSend(t *testing.T, ws *websocket.Conn, data interface{}) {
	if err := websocket.JSON.Send(ws, data); err != nil {
		t.Error(err)
	}
}

func websocketExpectResponse(t *testing.T, ws *websocket.Conn, event string,
	data map[string]*regexp.Regexp) (msg *WebsocketMessage) {
	var resp map[string]interface{}
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
	return
}

func websocketExpectError(t *testing.T, ws *websocket.Conn, status string) {
	websocketExpectResponse(t, ws, "__error", map[string]*regexp.Regexp{
		"status": regexp.MustCompile("^" + status + "$"),
	})
}

func backendDial(t *testing.T) net.Conn {
	c, err := net.Dial("tcp", "127.0.0.1:9081")
	c.SetReadTimeout((5 * time.Second).Nanoseconds())
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
	return c
}

func backendSend(t *testing.T, c net.Conn, frames ...string) {
	payload := strings.Join(frames, "\n")
	payload += "\n\r\n\r\n"
	_, err := c.Write([]byte(payload))
	if err != nil {
		t.Error(err)
	}
}

func backendExpectResponse(t *testing.T, c net.Conn, cmd string,
	frames ...string) {
	var msg = []string{}
	var buf = bufio.NewReader(c)
	var possibleEom = false
	for {
		chunk, err := buf.ReadSlice('\n')
		if err != nil {
			t.Error(err)
			return
		}
		if string(chunk) == "\r\n" {
			if possibleEom {
				break
			}
			possibleEom = true
			continue
		} else {
			possibleEom = false
		}
		msg = append(msg[:], string(chunk[:len(chunk)-1]))
	}
	if len(msg) < len(frames) + 1 {
		t.Errorf("Not enough frames to check")
	}
	if msg[0] != cmd {
		t.Errorf("Expected command to be '%s', got '%s'", cmd, msg[0])
	}
	for i, frame := range frames {
		if frame != msg[i+1] {
			t.Errorf("Expected frame to be '%s', got '%s'", frame, msg[i+1])
		}
	}
}

func backendExpectError(t *testing.T, c net.Conn, err int) {
	backendExpectResponse(t, c, "ER", fmt.Sprintf("%d", err))
}

func backendIdty() string {
	sid, _ := uuid.NewV4()
	return fmt.Sprintf("req:/test:%s:%s", v.accessToken, sid.String())
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

func testWebsocketAuthenticationWithoutToken(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"auth": map[string]interface{}{
			"foo": "bar",
		},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketAuthenticationWithInvalidTokenFormat(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"auth": map[string]interface{}{
			"token": map[string]interface{}{},
		},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketAuthenticationWithInvalidToken(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"auth": map[string]interface{}{
			"token": "invalid",
		},
	})
	websocketExpectError(t, ws, "Unauthorized")
}

func testWebsocketAuthenticationWithValidToken(t *testing.T,
	ws *websocket.Conn) {
	v, _ := ctx.Vhost("/test")
	token := v.GenerateSingleAccessToken(".*")
	websocketSend(t, ws, map[string]interface{}{
		"auth": map[string]interface{}{
			"token": token,
		},
	})
	websocketExpectResponse(t, ws, "__authenticated", nil)
}

func testWebsocketSubscribeWithoutChannelName(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketSubscribeWithEmptyChannelName(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "",
		},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketSubscribeWithInvalidChannelName(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "#&*^&^&&",
		},
	})
	websocketExpectError(t, ws, "Channel not found")
}

func testWebsocketSubscribeToNotExistingChannel(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "not-exists",
		},
	})
	websocketExpectError(t, ws, "Channel not found")
}

func testWebsocketSubscribeToPublicChannel(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "test",
		},
	})
	websocketExpectResponse(t, ws, "__subscribed",
		map[string]*regexp.Regexp{
			"channel": regexp.MustCompile("^test$"),
		})
}

func testWebsocketSubscribeToPrivateChannelWithoutAuthentication(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "private-test",
		},
	})
	websocketExpectError(t, ws, "Forbidden")
}

func testWebsocketSubscribeToPresenceChannelWithoutAuthentication(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "presence-test",
		},
	})
	websocketExpectError(t, ws, "Forbidden")
}

func testWebsocketSubscribeToPrivateChannelWithAuthentication(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "private-test",
		},
	})
	websocketExpectResponse(t, ws, "__subscribed", map[string]*regexp.Regexp{
		"channel": regexp.MustCompile("^private-test$"),
	})
}

func testWebsocketSubscribeToPresenceChannelWithAuthentication(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"subscribe": map[string]interface{}{
			"channel": "presence-test",
		},
	})
	websocketExpectResponse(t, ws, "__subscribed", map[string]*regexp.Regexp{
		"channel": regexp.MustCompile("^presence-test$"),
	})
	websocketExpectResponse(t, ws, "__memberJoined", nil)
}

func testWebsocketUnsubscribeWithoutChannelName(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"unsubscribe": map[string]interface{}{},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketUnsubscribeWithEmptyChannelName(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"unsubscribe": map[string]interface{}{
			"channel": "",
		},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketUnsubscribeWithInvalidChannelName(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"unsubscribe": map[string]interface{}{
			"channel": "#&*^&^&&",
		},
	})
	websocketExpectError(t, ws, "Channel not found")
}

func testWebsocketUnsubscribeNotSubscribedChannel(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"unsubscribe": map[string]interface{}{
			"channel": "presence-test",
		},
	})
	websocketExpectError(t, ws, "Not subscribed")
}

func testWebsocketUnsubscribeSubscribedChannel(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"unsubscribe": map[string]interface{}{
			"channel": "test",
		},
	})
	websocketExpectResponse(t, ws, "__unsubscribed",
		map[string]*regexp.Regexp{
			"channel": regexp.MustCompile("^test$"),
		})
}

func testWebsocketBroadcastWithoutChannelSpecified(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"event": "hello",
			"data":  map[string]interface{}{},
		},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketBroadcastWithEmptyChannelSpecified(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "",
			"event":   "hello",
			"data":    map[string]interface{}{},
		},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketBroadcastWithoutEventSpecified(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "test",
			"data":    map[string]interface{}{},
		},
	})
	websocketExpectError(t, ws, "Bad request")
}

func testWebsocketBroadcastToNotSubscribedChannel(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "presence-test",
			"event":   "hello",
			"data":    map[string]interface{}{},
		},
	})
	websocketExpectError(t, ws, "Not subscribed")
}

func testWebsocketBroadcastToNotExistingChannel(t *testing.T,
	ws *websocket.Conn) {
	websocketSend(t, ws, map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "not-exists",
			"event":   "hello",
			"data":    map[string]interface{}{},
		},
	})
	websocketExpectError(t, ws, "Channel not found")
}

func testWebsocketBroadcast(t *testing.T, wss []*websocket.Conn) {
	websocketSend(t, wss[0], map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "test",
			"event":   "hello",
			"data":    map[string]interface{}{"foo": "bar"},
		},
	})
	for _, ws := range wss {
		websocketExpectResponse(t, ws, "hello", map[string]*regexp.Regexp{
			"sid":     regexp.MustCompile("^.{36}$"),
			"channel": regexp.MustCompile("^test$"),
			"foo":     regexp.MustCompile("^bar$"),
		})
	}
}

func testWebsocketPresenceChannelSubscribeBehaviour(t *testing.T,
	wss []*websocket.Conn) {
	for i := range wss {
		websocketSend(t, wss[i], map[string]interface{}{
			"subscribe": map[string]interface{}{
				"channel": "presence-test",
				"data":    map[string]interface{}{"foo": "bar"},
			},
		})
		msg := websocketExpectResponse(t, wss[i], "__subscribed",
			map[string]*regexp.Regexp{
				"channel": regexp.MustCompile("^presence-test$"),
			})
		subscribers, ok := msg.Get("subscribers").([]interface{})
		if !ok || len(subscribers) != i {
			t.Errorf("Expected to get valid list of subscribers, got %v", subscribers)
		}
		for j := range wss[:i+1] {
			websocketExpectResponse(t, wss[j], "__memberJoined",
				map[string]*regexp.Regexp{
					"sid":     regexp.MustCompile("^.{36}"),
					"channel": regexp.MustCompile("^presence-test$"),
					"foo":     regexp.MustCompile("^bar$"),
				})
		}
	}
}

func testWebsocketPresenceChannelUnsubscribeBehaviour(t *testing.T,
	wss []*websocket.Conn) {
	for i := range wss {
		websocketSend(t, wss[i], map[string]interface{}{
			"unsubscribe": map[string]interface{}{
				"channel": "presence-test",
				"data":    map[string]interface{}{"bar": "foo"},
			},
		})
		websocketExpectResponse(t, wss[i], "__unsubscribed",
			map[string]*regexp.Regexp{
				"channel": regexp.MustCompile("^presence-test$"),
			})
		for j := range wss[i+1:] {
			websocketExpectResponse(t, wss[j+i+1], "__memberLeft",
				map[string]*regexp.Regexp{
					"sid":     regexp.MustCompile("^.{36}"),
					"channel": regexp.MustCompile("^presence-test$"),
					"bar":     regexp.MustCompile("^foo$"),
				})
		}
	}
}

func testBackendBadRequest(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, "bad request")
	backendExpectError(t, c, 400)
}

func testBackendBadIdentity(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, "bad identity", "", "OC", "test")
	backendExpectError(t, c, 402)
}

func testBackendOpenChannelWithoutName(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "OC")
	backendExpectError(t, c, 400)
}

func testBackendOpenChannelWithInvalidName(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "OC", "%f%df")
	backendExpectError(t, c, 451)
}

func testBackendOpenExistingChannel(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "OC", "test")
	backendExpectResponse(t, c, "OK")
}

func testBackendOpenNewChannel(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "OC", "new-test")
	backendExpectResponse(t, c, "OK")
	ch, err := v.Channel("new-test")
	if err != nil || ch == nil {
		t.Errorf("Expected to open new channel")
	}
}

func testBackendCloseNotExistingChannel(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "CC", "not-exists")
	backendExpectError(t, c, 454)
}

func testBackendCloseChannelWithoutName(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "CC")
	backendExpectError(t, c, 400)
}

func testBackendCloseChannelWithInvalidName(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "CC", "%dsdf%")
	backendExpectError(t, c, 454)
}

func testBackendRequestSingleAccessTokenWithoutPermission(t *testing.T,
	c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "AT")
	backendExpectError(t, c, 400)
}

func testBackendRequestSingleAccessTokenWithInvalidPermission(t *testing.T,
	c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "AT", "..*{34}")
	backendExpectError(t, c, 597)
}

func testBackendRequestSingleAccessTokenWithValidPermission(t *testing.T,
	c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "AT", "(foo|bar)")
	<-time.After(1e6)
	var token string
	for t, _ := range v.permissions {
		token = t
		break
	}
	if token == "" {
		t.Errorf("Expected to generate single access token")
	}
	backendExpectResponse(t, c, "AT", token)
}

func testBackendBroadcast(t *testing.T, c net.Conn, wss []*websocket.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "BC", "test", "hello", "{\"foo\":\"bar\"}")
	backendExpectResponse(t, c, "OK")
	for _, ws := range wss {
		websocketExpectResponse(t, ws, "hello", map[string]*regexp.Regexp{
			"channel": regexp.MustCompile("^test$"),
			"foo":     regexp.MustCompile("^bar$"),
		})
	}
}

func testBackendBroadcastWithEmptyChannelName(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "BC", "", "hello", "{\"foo\":\"bar\"}")
	backendExpectError(t, c, 400)
}

func testBackendBroadcastWithEmptyEventName(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "BC", "test", "", "{\"foo\":\"bar\"}")
	backendExpectError(t, c, 400)
}

func testBackendBroadcastToNotExistingChannel(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "BC", "not-exists", "hello", "{\"foo\":\"bar\"}")
	backendExpectError(t, c, 454)
}

func testBackendBroadcastWithInvalidData(t *testing.T, c net.Conn) {
	c = backendDial(t)
	backendSend(t, c, backendIdty(), "", "BC", "test", "hello", "&*&*")
	backendExpectResponse(t, c, "OK")
}

func TestAllTheThings(t *testing.T) {
	var ws *websocket.Conn
	var req net.Conn
	var wss [5]*websocket.Conn

	ws = websocketDial(t)
	testWebsocketConnect(t, ws)
	testWebsocketBadRequests(t, ws)
	testWebsocketAuthenticationWithoutToken(t, ws)
	testWebsocketAuthenticationWithInvalidTokenFormat(t, ws)
	testWebsocketAuthenticationWithInvalidToken(t, ws)
	testWebsocketAuthenticationWithValidToken(t, ws)
	testWebsocketSubscribeWithoutChannelName(t, ws)
	testWebsocketSubscribeWithEmptyChannelName(t, ws)
	testWebsocketSubscribeWithInvalidChannelName(t, ws)
	testWebsocketSubscribeToNotExistingChannel(t, ws)
	testWebsocketUnsubscribeWithoutChannelName(t, ws)
	testWebsocketUnsubscribeWithEmptyChannelName(t, ws)
	testWebsocketUnsubscribeWithInvalidChannelName(t, ws)
	testWebsocketUnsubscribeNotSubscribedChannel(t, ws)
	ws.Close()

	ws = websocketDial(t)
	testWebsocketConnect(t, ws)
	testWebsocketSubscribeToPublicChannel(t, ws)
	testWebsocketUnsubscribeSubscribedChannel(t, ws)
	testWebsocketSubscribeToPrivateChannelWithoutAuthentication(t, ws)
	testWebsocketSubscribeToPresenceChannelWithoutAuthentication(t, ws)
	testWebsocketAuthenticationWithValidToken(t, ws)
	testWebsocketSubscribeToPrivateChannelWithAuthentication(t, ws)
	testWebsocketSubscribeToPresenceChannelWithAuthentication(t, ws)
	ws.Close()

	for i := range wss {
		wss[i] = websocketDial(t)
		testWebsocketConnect(t, wss[i])
		testWebsocketAuthenticationWithValidToken(t, wss[i])
	}
	testWebsocketPresenceChannelSubscribeBehaviour(t, wss[:])
	testWebsocketPresenceChannelUnsubscribeBehaviour(t, wss[:])
	for i := range wss {
		wss[i].Close()
		wss[i] = nil
	}

	ws = websocketDial(t)
	testWebsocketConnect(t, ws)
	testWebsocketBroadcastWithoutChannelSpecified(t, ws)
	testWebsocketBroadcastWithEmptyChannelSpecified(t, ws)
	testWebsocketBroadcastWithoutEventSpecified(t, ws)
	testWebsocketBroadcastToNotSubscribedChannel(t, ws)
	testWebsocketBroadcastToNotExistingChannel(t, ws)
	ws.Close()

	for i := range wss {
		wss[i] = websocketDial(t)
		testWebsocketConnect(t, wss[i])
		testWebsocketSubscribeToPublicChannel(t, wss[i])
	}
	testWebsocketBroadcast(t, wss[:])
	testBackendBroadcast(t, req, wss[:])
	for i := range wss {
		wss[i].Close()
		wss[i] = nil
	}

	testBackendBadRequest(t, req)
	testBackendBadIdentity(t, req)
	testBackendOpenChannelWithoutName(t, req)
	testBackendOpenChannelWithInvalidName(t, req)
	testBackendOpenExistingChannel(t, req)
	testBackendOpenNewChannel(t, req)
	testBackendCloseChannelWithoutName(t, req)
	testBackendCloseChannelWithInvalidName(t, req)
	testBackendCloseNotExistingChannel(t, req)
	testBackendRequestSingleAccessTokenWithoutPermission(t, req)
	testBackendRequestSingleAccessTokenWithInvalidPermission(t, req)
	testBackendRequestSingleAccessTokenWithValidPermission(t, req)
	testBackendBroadcastWithEmptyChannelName(t, req)
	testBackendBroadcastWithEmptyEventName(t, req)
	testBackendBroadcastToNotExistingChannel(t, req)
	testBackendBroadcastWithInvalidData(t, req)
}

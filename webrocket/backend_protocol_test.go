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
	zmq "../gozmq"
	uuid "../uuid"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"
	"websocket"
)

var (
	req  zmq.Socket
	zctx zmq.Context
	err  error
	be   Endpoint
	bv   *Vhost
)

func init() {
	ctx := NewContext()
	ctx.SetLog(log.New(bytes.NewBuffer([]byte{}), "", log.LstdFlags))
	we := ctx.NewWebsocketEndpoint("", 9773)
	be = ctx.NewBackendEndpoint("", 9772)
	bv, _ = ctx.AddVhost("/test")
	bv.OpenChannel("test")
	go be.ListenAndServe()
	go we.ListenAndServe()
	zctx, _ = zmq.NewContext()
	req, err = zctx.NewSocket(zmq.REQ)
}

func breqrecv(t *testing.T) *Message {
	data, err := req.Recv(0)
	if err != nil {
		t.Error(err)
		return nil
	}
	msg, err := newMessageFromJSON(data)
	if err != nil {
		t.Error(err)
	}
	return msg
}

func berr(msg *Message) string {
	s, _ := msg.Get("status").(string)
	return s
}

func bokstatus(msg *Message) string {
	if msg.Event() != "__ok" {
		return ""
	}
	s, _ := msg.Get("status").(string)
	return s
}

func doTestBackendReqConnectWithInvalidIdentitiy(t *testing.T) {
	req.SetSockOptString(zmq.IDENTITY, "invalid")
	req.Connect("tcp://127.0.0.1:9772")
	req.Send([]byte("{}"), 0)
	resp := breqrecv(t)
	if berr(resp) != "Unauthorized" {
		t.Errorf("Expected to be unauthorized")
	}
}

func doTestBackendReqConnectWithValidIdentity(t *testing.T) {
	req.Close()
	req, _ = zctx.NewSocket(zmq.REQ)
	uuid, _ := uuid.NewV4()
	req.SetSockOptString(zmq.IDENTITY, fmt.Sprintf("req:/test:%s:%s", bv.accessToken, uuid.String()))
	req.Connect("tcp://127.0.0.1:9772")
	req.Send([]byte("{}"), 0)
	resp := breqrecv(t)
	if berr(resp) == "Unauthorized" {
		t.Errorf("Expected to be authenticated")
	}
}

func doTestBackendReqRequestWithInvalidData(t *testing.T) {
	req.Send([]byte("[\"invalid\"]"), 0)
	resp := breqrecv(t)
	if berr(resp) != "Bad request" {
		t.Errorf("Expeted a bad request error")
	}
}

func doTestBackendReqRequestWithNotFoundCommand(t *testing.T) {
	req.Send([]byte("{\"notfound\": {}}"), 0)
	resp := breqrecv(t)
	if berr(resp) != "Bad request" {
		t.Errorf("Expeted a bad request error")
	}
}

func doTestBackendReqOpenChannelWithNoNameSpecified(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"openChannel": map[string]interface{}{},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if berr(resp) != "Bad request" {
		t.Errorf("Expected a bad request error")
	}
}

func doTestBackendReqOpenChannelWithInvalidNameFormat(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"openChannel": map[string]interface{}{
			"channel": map[string]interface{}{},
		},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if berr(resp) != "Bad request" {
		t.Errorf("Expected a bad request error")
	}
}

func doTestBackendReqOpenChannelWithInvalidName(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"openChannel": map[string]interface{}{
			"channel": "this%name@#@is%invalid",
		},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if berr(resp) != "Invalid channel name" {
		t.Errorf("Expected a invalid channel name error")
	}
}

func doTestBackendReqOpenAlreadyExistingChannel(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"openChannel": map[string]interface{}{
			"channel": "test",
		},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if bokstatus(resp) != "Channel exists" {
		t.Errorf("Expected to get proper ok response")
	}
}

func doTestBackendReqOpenNewChannel(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"openChannel": map[string]interface{}{
			"channel": "test2",
		},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if bokstatus(resp) != "Channel opened" {
		t.Errorf("Expected to get proper ok response")
	}
	_, err := bv.Channel("test2")
	if err != nil {
		t.Errorf("Expected to open the specified channel")
	}
}

func doTestBackendReqCloseChannelWithNoNameSpecified(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"closeChannel": map[string]interface{}{},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if berr(resp) != "Bad request" {
		t.Errorf("Expected a bad request error")
	}
}

func doTestBackendReqCloseChannelWithInvalidNameFormat(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"closeChannel": map[string]interface{}{
			"channel": map[string]interface{}{},
		},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if berr(resp) != "Bad request" {
		t.Errorf("Expected a bad request error")
	}
}

func doTestBackendReqCloseNotExistingChannel(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"closeChannel": map[string]interface{}{
			"channel": "notexists",
		},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if berr(resp) != "Channel not found" {
		t.Errorf("Expected a channel not found error")
	}
}

func doTestBackendReqCloseChannel(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"closeChannel": map[string]interface{}{
			"channel": "test2",
		},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if bokstatus(resp) != "Channel closed" {
		t.Errorf("Expected to get proper ok response")
	}
	_, err := bv.Channel("test2")
	if err == nil {
		t.Errorf("Expected to close the specified channel")
	}
}

func doTestBackendReqRequestSingleAccessTokenWithoutSpecifyingPermissions(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"singleAccessToken": map[string]interface{}{},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if resp.Event() != "__singleAccessToken" {
		t.Errorf("Expected to get single access token back")
	}
	token, ok := resp.Get("token").(string)
	if !ok || len(token) != 128 {
		t.Errorf("Expected to get single access token back")
	}
	permission, ok := bv.ValidateSingleAccessToken(token)
	if !ok || permission.pattern != ".*" {
		t.Errorf("Expected to get valid single access token")
	}
}

func doTestBackendReqRequestSingleAccessTokenWithSpecifyingPermissions(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"singleAccessToken": map[string]interface{}{
			"permission": "(foo|bar)",
		},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if resp.Event() != "__singleAccessToken" {
		t.Errorf("Expected to get single access token back")
	}
	token, ok := resp.Get("token").(string)
	if !ok || len(token) != 128 {
		t.Errorf("Expected to get single access token back")
	}
	permission, ok := bv.ValidateSingleAccessToken(token)
	if !ok || permission.pattern != "(foo|bar)" {
		t.Errorf("Expected to get valid single access token")
	}
}

func doTestBackendReqBroadcastWhenInvalidChannelGiven(t *testing.T) {
	payload, _ := json.Marshal(map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "notexists", "event": "foo", "data": map[string]interface{}{},
		},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if berr(resp) != "Channel not found" {
		t.Errorf("Expected a channel not found error")
	}
}

func doTestBackendReqBroadcastValidData(t *testing.T) {
	var wss [2]*websocket.Conn
	for i := range wss {
		wss[i], _ = wsdial(9773)
		wsrecvfrom(t, wss[i])
		wssendto(t, wss[i], map[string]interface{}{
			"subscribe": map[string]interface{}{"channel": "test"},
		})
		wsrecvfrom(t, wss[i])
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"broadcast": map[string]interface{}{
			"channel": "test",
			"event":   "hello",
			"data":    map[string]interface{}{"foo": "bar"},
		},
	})
	req.Send(payload, 0)
	resp := breqrecv(t)
	if bokstatus(resp) != "Broadcasted" {
		t.Errorf("Expected to broadcast without errors")
	}
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
		foo, _ := resp.Get("foo").(string)
		if foo != "bar" {
			t.Errorf("Expected to broadcast passed data")
		}
	}
}

func TestBackendReqProtocol(t *testing.T) {
	// The same as with websocket protocol, some steps depends on others
	// so we need to preserve correct order.
	// FIXME: find better way to do it
	doTestBackendReqConnectWithInvalidIdentitiy(t)
	doTestBackendReqConnectWithValidIdentity(t)
	doTestBackendReqRequestWithInvalidData(t)
	doTestBackendReqRequestWithNotFoundCommand(t)
	doTestBackendReqOpenChannelWithNoNameSpecified(t)
	doTestBackendReqOpenChannelWithInvalidNameFormat(t)
	doTestBackendReqOpenChannelWithInvalidName(t)
	doTestBackendReqOpenAlreadyExistingChannel(t)
	doTestBackendReqOpenNewChannel(t)
	doTestBackendReqCloseChannelWithNoNameSpecified(t)
	doTestBackendReqCloseChannelWithInvalidNameFormat(t)
	doTestBackendReqCloseNotExistingChannel(t)
	doTestBackendReqCloseChannel(t)
	doTestBackendReqRequestSingleAccessTokenWithoutSpecifyingPermissions(t)
	doTestBackendReqRequestSingleAccessTokenWithSpecifyingPermissions(t)
	doTestBackendReqBroadcastWhenInvalidChannelGiven(t)
	doTestBackendReqBroadcastValidData(t)
}
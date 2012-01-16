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
/*
import (
	uuid "../uuid"
	"bytes"
	"fmt"
	"log"
	"testing"
	"time"
	"websocket"
	"net"
)

var (
	req *backendConnection
	err error
	be  Endpoint
	bv  *Vhost
)

func init() {
	ctx := NewContext()
	ctx.SetLog(log.New(bytes.NewBuffer([]byte{}), "", log.LstdFlags))
	we := ctx.NewWebsocketEndpoint(":9773")
	be = ctx.NewBackendEndpoint(":9772")
	bv, _ = ctx.AddVhost("/test")
	bv.OpenChannel("test")
	go be.ListenAndServe()
	go we.ListenAndServe()
	conn, err = conn.Dial("tcp", "127.0.0.1:9772")
	req = newBackendConnection(be, conn)
}

func breqrecv(t *testing.T) *backendRequest {
	data, err := req.Recv()
	if err != nil {
		t.Error(err)
		return nil
	}
	return data.msg
}

func breqsend(t *testing.T, idty, cmd, frames ...string) {
	err := req.Send(cmd, frames...)
	if err != nil {
		t.Error(err)
	}
}

func berr(req *backendRequest) string {
	if req.cmd != "ER" || len(req.msg) < 1 {
		return ""
	}
	return string(req.msg[0])
}

func bokstatus(req *backendRequest) bool {
	return req.cmd != "OK"
}

func doTestBackendReqConnectWithInvalidIdentitiy(t *testing.T) {
	breqsend(t, """{}")
	resp := breqrecv(t)
	if berr(resp) != "402" {
		t.Errorf("Expected to be unauthorized")
	}
}

func doTestBackendReqConnectWithValidIdentity(t *testing.T) {
	req.Close()
	req, _ = zctx.NewSocket(zmq.REQ)
	uuid, _ := uuid.NewV4()
	req.SetSockOptString(zmq.IDENTITY, fmt.Sprintf("req:/test:%s:%s", bv.accessToken, uuid.String()))
	req.Connect("tcp://127.0.0.1:9772")
	breqsend(t, "{}")
	resp := breqrecv(t)
	if berr(resp) == "402" {
		t.Errorf("Expected to be authenticated")
	}
}

func doTestBackendReqRequestWithInvalidCommand(t *testing.T) {
	breqsend(t, "invalid")
	resp := breqrecv(t)
	if berr(resp) != "400" {
		t.Errorf("Expeted a bad request error")
	}
}

func doTestBackendReqOpenChannelWithNoNameSpecified(t *testing.T) {
	breqsend(t, "OC", "")
	resp := breqrecv(t)
	if berr(resp) != "400" {
		t.Errorf("Expected a bad request error")
	}
}

func doTestBackendReqOpenChannelWithInvalidName(t *testing.T) {
	breqsend(t, "OC", "this%name@is$invalid")
	resp := breqrecv(t)
	if berr(resp) != "451" {
		t.Errorf("Expected an invalid channel name error")
	}
}

func doTestBackendReqOpenAlreadyExistingChannel(t *testing.T) {
	breqsend(t, "OC", "test")
	resp := breqrecv(t)
	if !bokstatus(resp) {
		t.Errorf("Expected to get proper ok response")
	}
}

func doTestBackendReqOpenNewChannel(t *testing.T) {
	breqsend(t, "OC", "test2")
	resp := breqrecv(t)
	if !bokstatus(resp) {
		t.Errorf("Expected to get proper ok response")
	}
	_, ok := bv.Channel("test2")
	if !ok {
		t.Errorf("Expected to open the specified channel")
	}
}

func doTestBackendReqCloseChannelWithNoNameSpecified(t *testing.T) {
	breqsend(t, "CC", "")
	resp := breqrecv(t)
	if berr(resp) != "400" {
		t.Errorf("Expected a bad request error")
	}
}

func doTestBackendReqCloseNotExistingChannel(t *testing.T) {
	breqsend(t, "CC", "notexists")
	resp := breqrecv(t)
	if berr(resp) != "454" {
		t.Errorf("Expected a channel not found error")
	}
}

func doTestBackendReqCloseChannel(t *testing.T) {
	breqsend(t, "CC", "test2")
	resp := breqrecv(t)
	if !bokstatus(resp) {
		t.Errorf("Expected to get proper ok response")
	}
	_, ok := bv.Channel("test2")
	if ok {
		t.Errorf("Expected to close the specified channel")
	}
}

func doTestBackendReqRequestSingleAccessTokenWithoutSpecifyingPermissions(t *testing.T) {
	breqsend(t, "AT")
	resp := breqrecv(t)
	if len(resp) < 2 || string(resp[0]) != "AT" {
		t.Errorf("Expected to get single access token back")
	}
	token := string(resp[1])
	if len(token) != 128 {
		t.Errorf("Expected to get single access token back")
	}
	permission, ok := bv.ValidateSingleAccessToken(token)
	if !ok || permission.Pattern != ".*" {
		t.Errorf("Expected to get valid single access token")
	}
}

func doTestBackendReqRequestSingleAccessTokenWithSpecifyingPermissions(t *testing.T) {
	breqsend(t, "AT", "(foo|bar)")
	resp := breqrecv(t)
	if len(resp) < 2 || string(resp[0]) != "AT" {
		t.Errorf("Expected to get single access token back")
	}
	token := string(resp[1])
	if len(token) != 128 {
		t.Errorf("Expected to get single access token back")
	}
	permission, ok := bv.ValidateSingleAccessToken(token)
	if !ok || permission.Pattern != "(foo|bar)" {
		t.Errorf("Expected to get valid single access token")
	}
}

func doTestBackendReqBroadcastWhenInvalidChannelGiven(t *testing.T) {
	breqsend(t, "BC", "node", "foo", "{}")
	resp := breqrecv(t)
	if berr(resp) != "454" {
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
	breqsend(t, "BC", "test", "hello", "{\"foo\": \"bar\"}")
	resp := breqrecv(t)
	if !bokstatus(resp) {
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
	doTestBackendReqRequestWithInvalidCommand(t)
	doTestBackendReqOpenChannelWithNoNameSpecified(t)
	doTestBackendReqOpenChannelWithInvalidName(t)
	doTestBackendReqOpenAlreadyExistingChannel(t)
	doTestBackendReqOpenNewChannel(t)
	doTestBackendReqCloseChannelWithNoNameSpecified(t)
	doTestBackendReqCloseNotExistingChannel(t)
	doTestBackendReqCloseChannel(t)
	doTestBackendReqRequestSingleAccessTokenWithoutSpecifyingPermissions(t)
	doTestBackendReqRequestSingleAccessTokenWithSpecifyingPermissions(t)
	doTestBackendReqBroadcastWhenInvalidChannelGiven(t)
	doTestBackendReqBroadcastValidData(t)
} 
*/
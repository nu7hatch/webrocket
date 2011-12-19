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
	"encoding/json"
	"errors"
	"fmt"
	zmq "../gozmq"
)

// BackendEndpoint is a wrapper for 0MQ ROUTER server. It handles
// all incoming connections from the backend application agents.
type BackendEndpoint struct {
	BaseEndpoint
	addr      string
	lobby     *backendLobby
	router    zmq.Socket
	zmqctx    zmq.Context
	isRunning bool
}

// newBackendEndpoint creates and preconfigures new backend server.
func (ctx *Context) NewBackendEndpoint(host string, port uint) Endpoint {
	if host == "" {
		host = "*"
	}
	e := &BackendEndpoint{}
	e.addr = fmt.Sprintf("tcp://%s:%d", host, int(port))
	e.ctx = ctx
	e.lobby = newBackendLobby()
	e.router = nil
	e.zmqctx, _ = zmq.NewContext() // XXX: should i handle this error here?
	ctx.backend = e
	return e
}

// Returns address to which this endpoint is bound.
func (w *BackendEndpoint) Addr() string {
	return w.addr
}

// Checks if the agent's identity is valid to access to specified
// vhost. If identity is approved, then returns in order: vhost
// to which agent is connected, its identity representation and
// boolean status. 
func (b *BackendEndpoint) authenticate(identity string) (v *Vhost, idty *agentIdentity, ok bool) {
	ok = false
	idty, err := parseBackendAgentIdentity(identity)
	if err == nil {
		// Invalid identity format
		return
	}
	vhost, err := b.ctx.Vhost(idty.Vhost)
	if err != nil {
		// Vhost doesn't exist
		return
	}
	if vhost.accessToken != idty.AccessToken {
		// Invalid access token
		return
	}
	ok = true
	return
}

// Server's event loop - handles all incoming messages.
func (b *BackendEndpoint) eventLoop() {
	b.alive()
	defer b.kill()
	for {
		if !b.IsRunning() {
			// TODO: send quit message to everyone
			break
		}
		b.receiveAndHandle()
	}
}

// receiveAndHandle performs single receive operation and dispatches
// received message.
func (b *BackendEndpoint) receiveAndHandle() {	
	// Receive the message blob
	aid, _ := b.router.Recv(zmq.SNDMORE)
	msg, _ := b.router.Recv(0)
	said := string(aid)
	// Authenticating an agent using it's identity...
	vhost, idty, ok := b.authenticate(said)
	if !ok {
		// If authentication failed, simply we can ignore him...
		return
	}
	// Dispatch the message depending on the socket type
	if idty.Type == zmq.DEALER {
		agent, ok := b.lobby.getAgentById(said)
		if !ok {
			// If it's first message from this agent, we have to
			// add him to the lobby.
			agent = newBackendAgent(b, vhost, aid)
			b.lobby.addAgent(agent)
		}
		println(msg)
		//dlrproto.dispatch(agent, msg)
	} else { // zmq.REQ
		//reqproto.dispatch(aid, msg)
	}
}

// Send enqueues specified message to internal lobby queue.
// Given message is load ballanced across all agents waiting
// in there.
func (b *BackendEndpoint) Send(payload interface{}) {
	b.lobby.enqueue(payload)
}

// SendTo directly sends message to specified agent.
func (b *BackendEndpoint) SendTo(id []byte, payload interface{}) (err error) {
	if b.router == nil {
		return errors.New("Endpoint is not running")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return
	}
	b.mtx.Lock()
	defer b.mtx.Unlock()
	err = b.router.SendMultipart([][]byte{id, encoded}, zmq.NOBLOCK)
	return
}

// ListenAndServe setups the 0MQ ROUTER socket and binds it to
// previously configured address.
func (b *BackendEndpoint) ListenAndServe() (err error) {
	b.router, err = b.zmqctx.NewSocket(zmq.ROUTER)
	defer b.router.Close()
	if err == nil {
		err = b.router.Bind(b.addr)
	}
	if err != nil {
		return
	}
	b.eventLoop()
	return
}

// TODO: ...
func (b *BackendEndpoint) ListenAndServeTLS(certFile, certKey string) (err error) {
	return errors.New("Not implemented")
}
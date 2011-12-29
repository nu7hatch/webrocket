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
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

// Global backend REQ protocol dispatcher.
var reqproto backendReqProtocol

// BackendEndpoint is a wrapper for 0MQ ROUTER server. It handles
// all incoming connections from the backend application agents.
type BackendEndpoint struct {
	BaseEndpoint
	addr      string
	lobbys    map[string]*backendLobby
	router    zmq.Socket
	zmqctx    zmq.Context
	isRunning bool
	log       *log.Logger
}

// newBackendEndpoint creates and preconfigures new backend server.
func (ctx *Context) NewBackendEndpoint(host string, port uint) Endpoint {
	if host == "" {
		host = "*"
	}
	e := &BackendEndpoint{}
	e.addr = fmt.Sprintf("tcp://%s:%d", host, int(port))
	e.ctx = ctx
	e.log = e.ctx.log
	e.lobbys = make(map[string]*backendLobby)
	e.router = nil
	e.zmqctx, _ = zmq.NewContext() // XXX: should i handle this error here?
	ctx.backend = e
	return e
}

// Returns address to which this endpoint is bound.
func (w *BackendEndpoint) Addr() string {
	return w.addr
}

// Registers a lobby for specified vhost. 
func (b *BackendEndpoint) registerVhost(vhost *Vhost) {
	b.lobbys[vhost.Path()] = newBackendLobby()
}

// Removes a lobby for specified vhost. 
func (b *BackendEndpoint) unregisterVhost(vhost *Vhost) {
	l, ok := b.lobbys[vhost.Path()]
	if !ok {
		return
	}
	l.kill()
	delete(b.lobbys, vhost.Path())
}

// Checks if the agent's identity is valid to access to specified
// vhost. If identity is approved, then returns in order: vhost
// to which agent is connected, its identity representation and
// boolean status. 
func (b *BackendEndpoint) authenticate(identity []byte) (vhost *Vhost,
	idty *backendIdentity, ok bool) {
	ok = false
	idty, err := parseBackendIdentity(identity)
	if err != nil {
		// Invalid identity format
		return
	}
	vhost, err = b.ctx.Vhost(idty.Vhost)
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
//
// XXX: handler is non-blocking, but there is no limit of the messages
// handled at the same time.
// TODO: Add such limit...
//
func (b *BackendEndpoint) receiveAndHandle() {
	// Receiving a message blob
	recv, err := b.router.RecvMultipart(0)
	if err != nil || len(recv) != 3 {
		return
	}
	aid, payload := recv[0], recv[2]
	// Non-blocking handler...
	go func() {
		// Authenticating an agent using it's identity...
		vhost, idty, ok := b.authenticate(aid)
		if !ok {
			// If authentication failed, sent an unauthorized error...
			// TODO: log error
			b.SendTo(aid, errorUnauthorized)
			return
		}
		// Parsing the message
		msg, err := newMessageFromJSON(payload)
		if err != nil {
			// TODO: log error
			b.SendTo(aid, errorBadRequest)
			return
		}
		// Dispatch the message depending on the socket type
		if idty.Type == zmq.DEALER {
			lobby, ok := b.lobbys[vhost.Path()]
			if !ok {
				// Something's fucked up, it should never happen
				// TODO: log error
				b.SendTo(aid, errorInternal)
				return
			}
			agent, ok := lobby.getAgentById(string(aid))
			if !ok {
				// If it's first message from this agent, we have to
				// add him to the lobby.
				agent = newBackendAgent(b, vhost, aid)
				lobby.addAgent(agent)
			}
			//dlrproto.dispatch(agent, msg)
		} else { // zmq.REQ
			reqproto.dispatch(b, vhost, aid, msg)
		}
	}()
}

// Send enqueues specified message to internal lobby queue.
// Given message is load ballanced across all agents waiting
// in there.
func (b *BackendEndpoint) Send(vhost *Vhost, payload interface{}) error {
	if vhost == nil {
		return errors.New("Invalid vhost")
	}
	lobby, ok := b.lobbys[vhost.Path()]
	if !ok {
		// Something's fucked...
		return errors.New("No lobby found for specified vhost")
	}
	lobby.enqueue(payload)
	return nil
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
	err = b.router.SendMultipart([][]byte{id, {}, encoded}, zmq.NOBLOCK)
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

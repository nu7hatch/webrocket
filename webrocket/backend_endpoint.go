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
	"errors"
	"fmt"
	"log"
	"sync"
	"net"
)

// BackendEndpoint is a wrapper for 0MQ ROUTER server. It handles
// all incoming connections from the backend application agents.
type BackendEndpoint struct {
	ctx       *Context
	addr      string
	alive     bool
	lobbys    map[string]*backendLobby
	router    zmq.Socket
	zmqctx    zmq.Context
	mtx       sync.Mutex
	log       *log.Logger
}

// newBackendEndpoint creates and preconfigures new backend server.
func (ctx *Context) NewBackendEndpoint(addr string) Endpoint {
	e := &BackendEndpoint{
		addr:   addr,
		ctx:    ctx,
		log:    ctx.log,
		lobbys: make(map[string]*backendLobby),
	}
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
	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.lobbys[vhost.Path()] = newBackendLobby()
}

// Removes a lobby for specified vhost. 
func (b *BackendEndpoint) unregisterVhost(vhost *Vhost) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	l, ok := b.lobbys[vhost.Path()]
	if !ok {
		return
	}
	l.Kill()
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
func (b *BackendEndpoint) serve() {
	for {
		if !b.IsAlive() {
			break
		}
		recv, err := b.router.RecvMultipart(0)
		if err != nil {
			// TODO: log
			continue
		}
		go b.handle(recv)
	}
}

// receiveAndHandle performs single receive operation and dispatches
// received message.
//
// XXX: handler is non-blocking, but there is no limit of the messages
// handled at the same time.
// TODO: Add such limit...
//
func (b *BackendEndpoint) handle(msg [][]byte) {
	var status string
	var code int
	// Check the message and read an envelope.
	if len(msg) < 3 {
		backendStatusLog(b, nil, "Bad request", 400, "")
		return
	}
	aid, cmd := msg[0], msg[2]
	request := newBackendRequest(b, nil, aid, string(cmd), msg[3:])
	// Authenticating an agent using it's identity...
	vhost, idty, ok := b.authenticate(aid)
	if ok {
		request.vhost = vhost
		if idty.Type == zmq.DEALER {
			status, code = backendDealerDispatch(request)
		} else { // zmq.REQ
			status, code = backendReqDispatch(request)
		}
	} else {
		status, code = "Unauthorized", 402
	}
	// Log status...
	if code >= 400 {
		backendError(b, vhost, aid, status, code, request.String())
	} else {
		backendStatusLog(b, vhost, status, code, request.String())
	}
}

// Trigger enqueues specified message to internal lobby queue.
// Given message is load ballanced across all agents waiting
// in there.
func (b *BackendEndpoint) Trigger(vhost *Vhost, payload interface{}) error {
	if vhost == nil {
		return errors.New("Invalid vhost")
	}
	lobby, ok := b.lobbys[vhost.Path()]
	if !ok {
		// Something's fucked, should never happen...
		return errors.New("No lobby found for specified vhost")
	}
	lobby.enqueue(payload)
	return nil
}

// SendTo directly sends message to specified agent.
func (b *BackendEndpoint) SendTo(id []byte, nonblock bool, cmd string, frames ...string) (err error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	if b.router == nil || !b.alive {
		return errors.New("Endpoint is not running")
	}
	var msg = make([][]byte, len(frames) + 3)
	msg[0], msg[2] = id, []byte(cmd)
	for i, frame := range frames {
		msg[i+3] = []byte(frame)
	}
	var flags zmq.SendRecvOption = 0
	if nonblock {
		flags |= zmq.NOBLOCK
	}
	err = b.router.SendMultipart(msg, flags) // FIXME: do polling here?
	return
}
	
	// ListenAndServe setups the 0MQ ROUTER socket and binds it to
// previously configured address.
func (b *BackendEndpoint) ListenAndServe() (err error) {
	b.router, err = b.zmqctx.NewSocket(zmq.ROUTER)
	if err != nil {
		return
	}
	defer b.router.Close()
	addr, err := net.ResolveTCPAddr("tcp", b.addr)
	if err != nil {
		return
	}
	host := addr.IP.String()
	if host == "<nil>" {
		host = "*"
	}
	err = b.router.Bind(fmt.Sprintf("tcp://%s:%d", host, addr.Port))
	if err != nil {
		return
	}
	b.alive = true
	b.serve()
	return
}

// TODO: ...
func (b *BackendEndpoint) ListenAndServeTLS(certFile, certKey string) (err error) {
	return errors.New("Not implemented")
}

// Returns true if this endpoint is activated.
func (w *BackendEndpoint) IsAlive() bool {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.alive
}

// Extended version of the kill func. Stops all alive agents.
func (w *BackendEndpoint) Kill() {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.alive = false
	for _, lobby := range w.lobbys {
		lobby.Kill()
	}
}
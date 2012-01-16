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
	"errors"
	"log"
	"sync"
	"net"
	"time"
)

// BackendEndpoint is a wrapper for 0MQ ROUTER server. It handles
// all incoming connections from the backend application agents.
type BackendEndpoint struct {
	ctx       *Context
	addr      string
	alive     bool
	lobbys    map[string]*backendLobby
	listener  *net.TCPListener
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
	for _, vhost := range ctx.vhosts {
		e.registerVhost(vhost)
	}
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

// receiveAndHandle performs single receive operation and dispatches
// received message.
func (b *BackendEndpoint) handle(conn net.Conn) (request *backendRequest, status string, code int) {
	bconn := newBackendConnection(b, conn)
	request, err := bconn.Recv()
	if err != nil {
		return nil, "Bad request", 400
	}
	println("REQ", request.cmd, string(request.id))
	// Authenticating an agent using it's identity...
	vhost, idty, ok := b.authenticate(request.id)
	if !ok {
		return request, "Unauthorized", 402
	}
	request.vhost = vhost
	if idty.Type == "dlr" { // DEALER
		var agent *BackendAgent
		lobby, ok := b.lobbys[vhost.Path()]
		if !ok {
			// Something's fucked up, it should never happen
			return request, "Internal error", 597
		}
		switch request.cmd {
		case "RD":
			// First message from the agent, means it's ready to work
			agent = newBackendAgent(bconn, request.vhost, request.id)
			lobby.addAgent(agent)
			// Blocking in here, keeping worker alive
			agent.listen()
			lobby.deleteAgent(agent)
			return nil, "Disconnected", 309
		case "HB":
			// Seems that agent sent heartbeat after liveness period,
			// we have to send a quit message restart it.
			request.Reply("QT")
			return request, "Expired", 408
		}
	} else { // REQ
		// Dispatching request to appropriate handler...
		handlerFunc, ok := backendReqProtocol[request.cmd]
		if ok {
			status, code = handlerFunc(request)
			return request, status, code
		}
	}
	return request, "Bad request", 400
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

// ListenAndServe setups the 0MQ ROUTER socket and binds it to
// previously configured address.
func (b *BackendEndpoint) ListenAndServe() (err error) {
	addr, err := net.ResolveTCPAddr("tcp", b.addr)
	if err != nil {
		return
	}
	b.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return
	}
	b.alive = true
	for {
		if !b.IsAlive() {
			break
		}
		var conn net.Conn
		conn, err = b.listener.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				log.Printf("accept error: %v\n", err)
				<-time.After(1 * time.Second)
				continue
			}
			return
		}
		go func(conn net.Conn) {
			request, status, code := b.handle(conn)
			if code >= 400 {
				backendError(b, request, status, code)
			} else if code < 300 {
				backendStatusLog(b, request, status, code)
			}
		}(conn)
	}
	return nil
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
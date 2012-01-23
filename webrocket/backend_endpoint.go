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
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

// The backend socket types.
const (
	BackendSocketReq    = "req"
	BackendSocketDealer = "dlr"
)

// BackendEndpoint implements a TCP server supporting the Backend Worker
// Protocol. It acts like a broker for both - backend clients (REQ) and
// workers (DEALER) using the majordomo pattern.
type BackendEndpoint struct {
	// Address to which this endpoint is bound.
	addr string
	// The parent context.
	ctx *Context
	// List of lobbys (handlers) for registered vhosts.
	lobbys *BackendLobbyMux
	// The underlaying TCP listener.
	listener *net.TCPListener
	// The endpoint's status.
	alive bool
	// Internal semaphore.
	mtx sync.Mutex
	// Internal logger.
	log *log.Logger
}

// Internal constructor
// -----------------------------------------------------------------------------

// newBackendEndpoint creates and preconfigures a new backend server endpoint.
//
// `ctx`  - The parent context.
// `addr` - The host and port to which this endpoint will be bound.
//
// Returns new configured backend endpoint.
func newBackendEndpoint(ctx *Context, addr string) *BackendEndpoint {
	return &BackendEndpoint{
		lobbys: NewBackendLobbyMux(),
		addr:   addr,
		ctx:    ctx,
		log:    ctx.log,
	}
}

// Internal
// -----------------------------------------------------------------------------

// registerVhost registers a new handler for the specified vhost. Not
// threadsafe, called only from the context's addVhost function.
//
// vhost - The vhost to be registered.
//
func (b *BackendEndpoint) registerVhost(vhost *Vhost) {
	l := newBackendLobby()
	b.lobbys.AddLobby(vhost.Path(), l)
}

// unregisterVhost removes a handler for the specified vhost if such has
// been registered before. Not threadsafe, called only from the context's
// deleteVhost function.
//
// vhost - The vhost to be removed.
//
func (b *BackendEndpoint) unregisterVhost(vhost *Vhost) {
	b.lobbys.DeleteLobby(vhost.Path())
}

// serve accepts the incoming TCP connections and runs a separate handler
// for each one.
//
// Returns an error if something went wrong.
func (b *BackendEndpoint) serve() (err error) {
	for {
		if !b.IsAlive() {
			break
		}
		var conn net.Conn
		if conn, err = b.listener.Accept(); err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				log.Printf("accept error: %v\n", err)
				<-time.After(1 * time.Second)
				continue
			}
			return
		}
		go b.handle(conn)
	}
	return
}

// authenticate checks if the worker's identity has access to the
// specified vhost.
//
// rawIdentity - The identity line to be parsed and authorized.
//
// If identity is approved, then returns in order: vhost to which worker is
// connected, its identity representation and boolean status.
func (b *BackendEndpoint) authenticate(rawIdentity string) (vhost *Vhost,
	idty *backendIdentity, ok bool) {
	var err error
	if idty, err = parseBackendIdentity(rawIdentity); err != nil {
		// Invalid identity format
		return
	}
	if vhost, err = b.ctx.Vhost(idty.Vhost); err != nil {
		// Vhost doesn't exist
		return
	}
	if vhost.accessToken != idty.AccessToken {
		// Invalid access token
		return
	}
	return vhost, idty, true
}

// handle gets the date from the specified connection and handles it in a way
// appropriate for the sender's socket type.
//
// conn - The TCP connection to be handled.
//
func (b *BackendEndpoint) handle(conn net.Conn) {
	var vhost *Vhost
	var req *backendRequest
	var idty *backendIdentity
	var ok bool
	var err error
	var status string
	var code int

	c := newBackendConnection(conn)
	if req, err = c.Recv(); err != nil {
		status, code = "Bad request", 400
		c.Send("ER", "400")
		goto log
	}
	if vhost, idty, ok = b.authenticate(req.Identity); !ok {
		status, code = "Unauthorized", 402
		goto log
	}
	// Dispatch the request...
	switch {
	case idty.Type == BackendSocketDealer:
		status, code = b.dispatchDealer(vhost, req, idty)
	case idty.Type == BackendSocketReq:
		status, code = b.dispatchReq(vhost, req, idty)
	default:
		status, code = "Bad request", 400
	}

log:
	b.logStatus(vhost, status, code, req)
}

// dispatchDealer handles a request received from the dealer socket.
//
// vhost - The vhost to which the message has been sent.
// req   - The request to be handled.
// idty  - The sender's identity.
//
// Returns a status message and code.
func (b *BackendEndpoint) dispatchDealer(vhost *Vhost, req *backendRequest,
	idty *backendIdentity) (string, int) {
	var worker *BackendWorker
	var lobby *backendLobby

	if lobby = b.lobbys.Match(vhost.Path()); lobby == nil {
		// Something's fucked up, it should never happen...
		return "Internal error", 597
	}
	switch req.Command {
	case "RD": // Ready
		worker = newBackendWorker(req.conn, idty.Id)
		lobby.addWorker(worker)
		defer lobby.deleteWorker(worker)
		// Blocking in here, keeping worker alive.
		worker.listen()
		return "Disconnected", 309
	case "HB": // Heartbeat
		// Seems that worker sent heartbeat after liveness period,
		// we have to send a quit message restart it.
		req.Reply("QT")
		return "Expired", 408
	}
	// Invalid command received...
	return "Bad request", 400
}

// dispatchReq handles a request received from the req socket.
//
// vhost - The vhost to which message has been sent.
// req   - The request to be handled.
// idty  - The sender's identity.
//
// Returns a status message and code.
func (b *BackendEndpoint) dispatchReq(vhost *Vhost, req *backendRequest,
	idty *backendIdentity) (status string, code int) {
	switch req.Command {
	case "BC": // Broadcast
		status, code = b.handleReqBroadcast(vhost, req)
	case "OC": // Open channel
		status, code = b.handleReqOpenChannel(vhost, req)
	case "CC": // Close channel
		status, code = b.handleReqCloseChannel(vhost, req)
	case "AT": // Generate single access token
		status, code = b.handleReqSingleAccessTokenRequest(vhost, req)
	default:
		status, code = "Bad request", 400
	}
	return
}

func (b *BackendEndpoint) logStatus(vhost *Vhost, status string, code int,
	req *backendRequest) {
	switch {
	case code >= 400:
		if req != nil {
			req.Reply("ER", strconv.Itoa(code))
		}
	case code >= 300 && code < 400:
		// Log information statuses only when debug mode is enabled.
		// TODO: log after adding a debug mode...
		return
	case code < 300:
		// Nothing to do, just go to logging...
	}
	vhostPath := "???"
	if vhost != nil {
		vhostPath = vhost.Path()
	}
	reqString := "[]"
	if req != nil {
		reqString = req.String()
	}
	b.log.Printf("backend[%s]: %d %s; %s", vhostPath, code, status, reqString)
}

// handleReqBroadcast is a handler for the backend's broadcast (BC) request.
//
// vhost - Related vhost.
// req   - The request to be handled.
//
// Returns textual status and code.
func (b *BackendEndpoint) handleReqBroadcast(vhost *Vhost, req *backendRequest) (string, int) {
	// <<<
	// channel name\n
	// event name\n
	// {...}\n
	// >>>
	var chanName, eventName string
	var data map[string]interface{}
	var channel *Channel
	var err error

	if req.Len() < 3 {
		return "Bad request", 400
	}
	chanName, eventName = string(req.Message[0]), string(req.Message[1])
	if chanName == "" || eventName == "" {
		// No channel or event name specified!
		return "Bad request", 400
	}
	if err = json.Unmarshal(req.Message[2], &data); err != nil {
		// No data specified, making empty one...
		data = make(map[string]interface{})
	}
	if channel, err = vhost.Channel(chanName); err != nil {
		// Request channel doesn't exist!
		return "Channel not found", 454
	}
	// Extending data with the channel name before pass it forward.
	data["channel"] = chanName
	channel.Broadcast(map[string]interface{}{eventName: data})
	req.Reply("OK")
	return "Broadcasted", 204
}

// handleReqOpenChannel is a handler for the backend's open channel (OC) request.
//
// vhost - Related vhost.
// req   - The request to be handled.
//
// Returns textual status and code.
func (b *BackendEndpoint) handleReqOpenChannel(vhost *Vhost, req *backendRequest) (string, int) {
	// <<<
	// channel name\n
	// >>>
	var chanName string
	var chanType ChannelType
	var err error

	if req.Len() < 1 {
		return "Bad request", 400
	}
	chanName = string(req.Message[0])
	if chanName == "" {
		// No channel name or type specified.
		return "Bad request", 400
	}
	if _, err = vhost.Channel(chanName); err == nil {
		// Channel with such name already exists, it's ok!
		req.Reply("OK")
		return "Channel exists", 251
	}
	chanType = channelTypeFromName(chanName)
	if _, err = vhost.OpenChannel(chanName, chanType); err != nil {
		// Requested channel name is invalid!
		return "Invalid channel name", 451
	}
	req.Reply("OK")
	return "Channel opened", 250
}

// handleReqCloseChannel is a handler for the backend's close channel (CC) request.
//
// vhost - Related vhost.
// req   - The request to be handled.
//
// Returns textual status and code.
func (b *BackendEndpoint) handleReqCloseChannel(vhost *Vhost, req *backendRequest) (string, int) {
	// <<<
	// channel name\n
	// >>>
	var chanName string
	var ok bool

	if req.Len() < 1 {
		return "Bad request", 400
	}
	if chanName = string(req.Message[0]); chanName == "" {
		// No channel name specified.
		return "Bad request", 400
	}
	if ok = vhost.DeleteChannel(chanName); !ok {
		return "Channel not found", 454
	}
	req.Reply("OK")
	return "Channel closed", 252
}

// handleReqSingleAccessTokenRequest is a handler for the backend's single
// access token (AT) request.
//
// vhost - Related vhost.
// req   - The request to be handled.
//
// Returns textual status and code.
func (b *BackendEndpoint) handleReqSingleAccessTokenRequest(vhost *Vhost,
	req *backendRequest) (string, int) {
	// <<<
	// permission regexp\n
	// >>>
	var pattern, token string

	if req.Len() < 1 {
		return "Bad request", 400
	}
	if pattern = string(req.Message[0]); pattern == "" {
		// No permission regexp specified.
		return "Bad request", 400
	}
	if token = vhost.GenerateSingleAccessToken(pattern); token == "" {
		// Couldn't generate an access token.
		return "Internal error", 597
	}
	req.Reply("AT", token)
	return "Single access token generated", 270
}

// Exported
// -----------------------------------------------------------------------------

// Addr returns an address to which this endpoint is bound.
func (w *BackendEndpoint) Addr() string {
	return w.addr
}

// Trigger enqueues the message in the internal lobby queue. Given message is
// load ballanced across all workers waiting in there.
//
// vhost   - Related vhost.
// payload - The payload to be enqueued.
//
// Returns an error if something went wrong.
func (b *BackendEndpoint) Trigger(vhost *Vhost, payload interface{}) error {
	if vhost == nil {
		return errors.New("invalid vhost")
	}
	lobby := b.lobbys.Match(vhost.Path())
	if lobby != nil {
		// Something's fucked up, should never happen...
		return errors.New("no lobby found for the specified vhost")
	}
	lobby.Enqueue(payload)
	return nil
}

// ListenAndServe setups endpoint's TCP listener for handling incoming
// backend worker's and client's connections.
func (b *BackendEndpoint) ListenAndServe() (err error) {
	addr, err := net.ResolveTCPAddr("tcp", b.addr)
	if err != nil {
		return
	}
	if b.listener, err = net.ListenTCP("tcp", addr); err != nil {
		return
	}
	b.alive = true
	return b.serve()
}

// TODO: ...
func (b *BackendEndpoint) ListenAndServeTLS(certFile, certKey string) (err error) {
	return errors.New("not implemented")
}

// IsAlive Returns whether the endpoint is alive or not.
func (w *BackendEndpoint) IsAlive() bool {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.alive && w.listener != nil
}

// Kill terminates execution of this endpoint, kills all registered lobbys
// and closes all open worker connections.
func (w *BackendEndpoint) Kill() {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	if w.alive && w.listener != nil {
		w.alive = false
		w.listener.Close()
		w.lobbys.KillAll()
	}
}

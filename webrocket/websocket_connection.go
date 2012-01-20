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
	"../uuid"
	"io"
	"sync"
	"websocket"
)

// WebsocketConnection represents a single WebSockets connection and
// implements an API for managing its subscriptions and authentication.
type WebsocketConnection struct {
	*websocket.Conn

	// Unique identifier of the connected client.
	id string
	// If authenticated, contains an access control information.
	permission *Permission
	// List of client's subscriptions
	subscriptions map[string]*Channel
	// Internal semaphore
	mtx sync.Mutex
}

// Internal constructor
// -----------------------------------------------------------------------------

// newWebsocketConnection wraps given WebSocket connection within the newly
// created WebsocketConnection structure.
//
// ws - The raw websocket connection to be wrapped.
//
// Returns wrapped websocket connection.
func newWebsocketConnection(ws *websocket.Conn) (c *WebsocketConnection) {
	uuid, _ := uuid.NewV4()
	c = &WebsocketConnection{
		Conn:          ws,
		id:            uuid.String(),
		subscriptions: make(map[string]*Channel),
	}
	// Send info that connection has been approved. Yeah,
	// Bruce Lee approves!
	c.Send(map[string]interface{}{
		"__connected": map[string]interface{}{
			"sid": c.Id(),
		},
	})
	return
}

// Internal
// -----------------------------------------------------------------------------

// authenticate marks the connection as authenticated by assigning given
// permissions information to it. Not threadsafe, used only from within
// websocket protocol's handlers which is blocking for specified connection.
//
// p - The permission information to be assigned to this connection.
//
func (c *WebsocketConnection) authenticate(p *Permission) {
	c.permission = p
	if p != nil {
		c.Send(map[string]interface{}{
			"__authenticated": map[string]interface{}{},
		})
	}
}

// reauthenticate cleans up previously set permissions, unsubscribes from
// all the channels and authenticates the client again using given permissions
// information.
//
// p - The permission information to be assigned to this connection.
//
func (c *WebsocketConnection) reauthenticate(p *Permission) {
	c.clearSubscriptions()
	c.authenticate(p)
}

// Removes all subscriptions created by this client.
func (c *WebsocketConnection) clearSubscriptions() {
	for _, ch := range c.subscriptions {
		ch.unsubscribe(c, map[string]interface{}{}, false)
	}
}

// Exported
// -----------------------------------------------------------------------------

// Id returns an unique session identifier attached to this connection.
func (c *WebsocketConnection) Id() string {
	return c.id
}

// IsAuthenticated returns whether this connection is authenticated or not.
// Not threadsafe, Used only from within websocket protocol's handlers which
// is blocking for specified connection.
func (c *WebsocketConnection) IsAuthenticated() bool {
	return c.permission != nil
}

// IsAllowed returns whether this connections is authenticated and has
// sufficient permissions to operate on a given channel. Not threadsafe,
// used only from within websocket protocol's handlers which is blocking
// for specified connection.
//
// channel - The channel to check permissions for.
//
func (c *WebsocketConnection) IsAllowed(channel string) bool {
	return c.IsAuthenticated() && c.permission.IsMatching(channel)
}

// Send serializes given payload with JSON and sends it to the client.
// Threadsafe, may be used from the websocket protocol's handlers and
// the channel's broadcaster.
//
// payload - A data to be send to the client.
//
func (c *WebsocketConnection) Send(payload interface{}) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.Conn == nil {
		return
	}
	if err := websocket.JSON.Send(c.Conn, payload); err != nil {
		// TODO: error log!
		//websocketStatusLog(c, "Not sent", 597, err.Error())
	}
}

// Receive reads a message from the client and parses it into the internal
// message object. If there is no data to read from the connection then it
// shall block until new data arrive. Not threadsafe, used only from within
// websocket handler's event loop.
//
// Returns message received from the connection.
func (c *WebsocketConnection) Receive() (*WebsocketMessage, error) {
	var recv map[string]interface{}
	if !c.IsAlive() {
		return nil, io.EOF
	}
	recv = make(map[string]interface{})
	if err := websocket.JSON.Receive(c.Conn, &recv); err != nil {
		return nil, err
	}
	return newWebsocketMessage(recv)
}

// IsAlive returns whether the connection is alive or not. Threadsafe, so far
// used internally only, but the related Kill function may be called from
// many goroutines.
func (c *WebsocketConnection) IsAlive() bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.Conn != nil
}

// Kill cleans up all subscriptions and closes the connection. This operation
// will mark connection as dead. Threadsafe, Used in the websocket endpoint
// and handlers. 
func (c *WebsocketConnection) Kill() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.clearSubscriptions()
	if c.Conn != nil {
		c.Conn.Close()
		c.Conn = nil
	}
}

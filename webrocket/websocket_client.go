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
	uuid "../uuid"
	"websocket"
)

// WebsocketClient represents single WebSockets connection.
type WebsocketClient struct {
	*connection
	*websocket.Conn
	id            string
	permission    *Permission
	subscriptions map[string]*Channel
}

// newWebsocketClient wraps given WebSocket connection within the newly created
// WebsocketClient structure. Each client uses separate goroutine to deal
// with the outgoing messages.
func newWebsocketClient(v *Vhost, ws *websocket.Conn) (c *WebsocketClient) {
	c = &WebsocketClient{Conn: ws}
	c.connection = newConnection(v)
	c.id = uuid.GenerateTime()
	c.subscriptions = make(map[string]*Channel)
	return c
}

// Marks this client as authenticate with given permissions. 
func (c *WebsocketClient) authenticate(p *Permission) {
	c.permission = p
}

// Returns true when given client is authenticated
func (c *WebsocketClient) IsAuthenticated() bool {
	return c.permission != nil
}

// Returns true if client is authenticated and its permissions
// allows to operate on specified channel.
func (c *WebsocketClient) IsAllowed(channel string) bool {
	return c.IsAuthenticated() && c.permission.IsMatching(channel)
}

// Returns an id of the current session.
func (c *WebsocketClient) Id() string {
	return c.id
}

// Sends specified payload to the client.
func (c *WebsocketClient) Send(payload interface{}) {
	if c.IsAlive() {
		c.mtx.Lock()
		err := websocket.JSON.Send(c.Conn, payload)
		c.mtx.Unlock()
		if err != nil {
			// Couldn't send to the client
			wsproto.log(c, "597", err.Error())
		}
	}
}

// Removes all subscriptions created by this client.
func (c *WebsocketClient) clearSubscriptions() {
	for _, ch := range c.subscriptions {
		ch.deleteSubscriber(c)
	}
}

// Kills the client and closes underlaying connection.
func (c *WebsocketClient) kill() {
	c.connection.kill()
	c.clearSubscriptions()
	c.Conn.Close()
}

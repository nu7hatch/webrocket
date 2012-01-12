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
	"time"
	"io"
	"encoding/json"
)

const (
	websocketClientDefaultMaxRetries = 3
	websocketClientDefaultRetryDelay = time.Duration(1e6)
)

// WebsocketClient represents single WebSockets connection.
type WebsocketClient struct {
	*connection
	*websocket.Conn
	id            string
	permission    *Permission
	subscriptions map[string]*Channel
	maxRetries    int
	retryDelay    time.Duration
}

// newWebsocketClient wraps given WebSocket connection within the newly created
// WebsocketClient structure. Each client uses separate goroutine to deal
// with the outgoing messages.
func newWebsocketClient(v *Vhost, ws *websocket.Conn) (c *WebsocketClient) {
	uuid, _ := uuid.NewV4()
	c = &WebsocketClient{
		Conn:          ws,
		id:            uuid.String(),
		maxRetries:    websocketClientDefaultMaxRetries,
		retryDelay:    websocketClientDefaultRetryDelay,
		connection:    newConnection(v),
		subscriptions: make(map[string]*Channel),
	}
	return
}

// Marks this client as authenticate with given permissions. 
func (c *WebsocketClient) authenticate(p *Permission) {
	c.permission = p
}

// Removes all subscriptions created by this client.
func (c *WebsocketClient) clearSubscriptions() {
	for _, ch := range c.subscriptions {
		ch.deleteSubscriber(c)
	}
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
		defer c.mtx.Unlock()
		err := websocket.JSON.Send(c.Conn, payload)
		if err != nil {
			websocketStatusLog(c, "Not send", 597, err.Error())
		}
	}
}

// Receive reads a message from the client and parses it into
// the internal message object. If there is no data to read from
// the connection then it block until new data arrive.
func (c *WebsocketClient) Receive() *Message {
again:
	if !c.IsAlive() {
		return nil
	}
	var recv map[string]interface{}
	err := websocket.JSON.Receive(c.Conn, &recv)
	if err != nil {
		if err == io.EOF {
			// End of file reached, which means that connection
			// has been closed.
			websocketError(c, "Connection closed", 598, "")
			c.Kill()
			return nil
		}
		websocketBadRequestError(c, "")
		goto again
	}
	msg, err := newMessage(recv)
	if err != nil {
		// Message couldn't be parsed due to invalid JSON format.
		msgstr, _ := json.Marshal(recv)
		websocketBadRequestError(c, string(msgstr))
		goto again
	}
	return msg
}

// Returns true if the connection is alive.
func (c *WebsocketClient) IsAlive() bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.Conn != nil
}

// Kills the client and closes underlaying connection.
func (c *WebsocketClient) Kill() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.clearSubscriptions()
	c.Conn.Close()
}

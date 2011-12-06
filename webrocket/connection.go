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
	"crypto/sha1"
	"fmt"
	"os"
	"websocket"
)

// generateUniqueToken creates unique token using system `/dev/urandom`.
func generateUniqueToken() string {
	f, _ := os.OpenFile("/dev/urandom", os.O_RDONLY, 0)
	b := make([]byte, 16)
	f.Read(b)
	f.Close()
	token := sha1.New()
	token.Write(b)
	return fmt.Sprintf("%x", token.Sum())
}

// Wrapper for standard websocket.Conn structure. Provides additional
// information about connection and maintains sessions. 
type conn struct {
	*websocket.Conn
	token    string
	session  *User
	vhost    *Vhost
	channels map[*Channel]bool
}

// wrapConn wraps standard websocket connection object into one
// adjusted for webrocket server funcionalities.
func wrapConn(ws *websocket.Conn, vhost *Vhost) *conn {
	c := &conn{Conn: ws, token: generateUniqueToken(), vhost: vhost}
	c.channels = make(map[*Channel]bool)
	return c
}

// A helper for quick sending encoded payloads to the connected client.
func (c *conn) send(data interface{}) error {
	err := c.vhost.codec.Send(c.Conn, data)
	if err != nil {
		c.vhost.Log.Printf("ws[%s]: ERR_NOT_SEND %s", c.vhost.path, err.Error())
	}
	return err
}

// Unsubscribes this client from all channels.
func (c *conn) unsubscribeAll() {
	for ch := range c.channels {
		ch.subscribe <- subscription{c, false}
	}
}

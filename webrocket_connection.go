// Copyright 2011 Chris Kowalik (nu7hatch). All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.
//
// Package webrocket implements advanced WebSocket server with custom
// protocols support. 
package webrocket

import (
	"os"
	"fmt"
	"errors"
	"websocket"
	"crypto/sha1"
)

// message is a simple structure which keeps the incoming events
// information and data.
type message struct {
	Event string
	Data  interface{}
}

// extractMessage converts received map into message structure. 
func NewMessage(data map[string]interface{}) (*message, error) {
	if len(data) != 1 {
		return nil, errors.New("Invalid message format")
	}
	msg := &message{}
	for k := range data {
		msg.Event = k
	}
	msg.Data = data[msg.Event]
	return msg, nil
}

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
	channels map[*channel]bool
}

// wrapConn wraps standard websocket connection object into one
// adjusted for webrocket server funcionalities.
func wrapConn(ws *websocket.Conn, vhost *Vhost) *conn {
	c := &conn{Conn: ws, token: generateUniqueToken(), vhost: vhost}
	c.channels = make(map[*channel]bool)
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
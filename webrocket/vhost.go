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
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"websocket"
)

// Vhost is an namespaced, standalone handler for websocket
// connections. Each vhost has it's own users and permission
// management setting, independent channels, etc.
type Vhost struct {
	Log         *log.Logger
	path        string
	isRunning   bool
	handler     websocket.Handler
	users       map[string]*User
	connections map[string]*wsConn
	channels    map[string]*Channel
	codec       websocket.Codec
	frontAPI    websocketAPI
}

// Returns new vhost configured to handle websocket connections.
func NewVhost(path string) *Vhost {
	v := &Vhost{path: path, isRunning: true}
	v.handler = websocket.Handler(func(ws *websocket.Conn) { v.handle(ws) })
	v.users = make(map[string]*User)
	v.connections = make(map[string]*wsConn)
	v.channels = make(map[string]*Channel)
	v.codec = websocket.JSON
	v.Log = log.New(os.Stderr, "", log.LstdFlags)
	return v
}

// Prepares new connection to enter in the event loop.
func (v *Vhost) handle(ws *websocket.Conn) {
	c := wrapWsConn(ws, v)
	v.connections[c.token] = c
	v.eventLoop(c)
	v.cleanup(c)
}

// cleanup removes all subscprionts and other relations
// between closed connection and the system.
func (v *Vhost) cleanup(c *wsConn) {
	c.unsubscribeAll()
	delete(v.connections, c.token)
}

// eventLoop maintains main loop for handled connection.
func (v *Vhost) eventLoop(c *wsConn) {
	for {
		if !v.IsRunning() {
			return
		}
		var recv map[string]interface{}
		err := v.codec.Receive(c.Conn, &recv)
		if err != nil {
			if err == io.EOF {
				return
			}
			v.frontAPI.Error(c, ErrInvalidDataReceived)
			v.Log.Printf("ws[%s]: ERR_INVALID_DATA_RECEIVED", v.path)
			continue
		}
		message, err := NewMessage(recv)
		if err != nil {
			v.frontAPI.Error(c, ErrInvalidMessageFormat)
			v.Log.Printf("ws[%s]: ERR_INVALID_MESSAGE_FORMAT", v.path)
			continue
		}
		keepgoing, _ := v.frontAPI.Dispatch(c, message)
		if !keepgoing {
			return
		}
	}
}

// Stop closes all connection handled by this vhost and stops
// its eventLoop.
func (v *Vhost) Stop() {
	v.isRunning = false
}

// Is this vhost running?
func (v *Vhost) IsRunning() bool {
	return v.isRunning
}

// Returns list of active connections.
func (v *Vhost) Connections() map[string]*wsConn {
	return v.connections
}

// Returns path name.
func (v *Vhost) Path() string {
	return v.path
}

// AddUser configures new user account within this vhost.
func (v *Vhost) AddUser(name, secret string, permission int) error {
	if len(name) == 0 {
		return errors.New("User name can't be blank")
	}
	_, ok := v.users[name]
	if ok {
		return errors.New("User already exists")
	}
	if permission == 0 {
		return errors.New("Invalid permissions")
	}
	v.users[name] = NewUser(name, secret, permission)
	v.Log.Printf("vhost[%s]: ADD_USER name='%s' permission=%d", v.path, name, permission)
	return nil
}

// DeleteUser deletes user account with given name.
func (v *Vhost) DeleteUser(name string) error {
	_, ok := v.users[name]
	if !ok {
		return errors.New("User doesn't exist")
	}
	delete(v.users, name)
	v.Log.Printf("vhost[%s]: DELETE_USER name='%s'", v.path, name)
	return nil
}

// SetUserPermissions configures user access.
func (v *Vhost) SetUserPermissions(name string, permission int) error {
	user, ok := v.users[name]
	if !ok {
		return errors.New("User doesn't exist")
	}
	if permission == 0 {
		return errors.New("Invalid permissions")
	}
	user.Permission = permission
	v.Log.Printf("vhost[%s]: SET_USER_PERMISSION name='%s' permission=%d", v.path, name, permission)
	return nil
}

// Returns list of configured user accounts.
func (v *Vhost) Users() map[string]*User {
	return v.users
}

// OpenChannel creates new channel ready to subscribe.
func (v *Vhost) CreateChannel(name string) *Channel {
	channel := NewChannel(v, name)
	v.channels[name] = channel
	v.Log.Printf("vhost[%s]: CREATE_CHANNEL name='%s'", v.path, name)
	return channel
}

// Returns specified channel.
func (v *Vhost) GetChannel(name string) (*Channel, bool) {
	channel, ok := v.channels[name]
	return channel, ok
}

// Returns specified channel. If channel doesn't exist, then will be
// created automatically.
func (v *Vhost) GetOrCreateChannel(name string) *Channel {
	channel, ok := v.channels[name]
	if !ok {
		return v.CreateChannel(name)
	}
	return channel
}

// Returns list of used channels.
func (v *Vhost) Channels() map[string]*Channel {
	return v.channels
}

// Returns specified user.
func (v *Vhost) GetUser(name string) (*User, bool) {
	user, ok := v.users[name]
	return user, ok
}

// ServeHTTP extends standard websocket.Handler implementation
// of http.Handler interface.
func (v *Vhost) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if v.isRunning {
		v.handler.ServeHTTP(w, req)
	}
}

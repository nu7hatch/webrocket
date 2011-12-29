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
	"crypto/sha1"
	"errors"
	"fmt"
	"regexp"
	"sync"
)

// Vhost name will be validated using this pattern.
const vhostNamePattern = "^/[\\w\\d\\-\\_]+(/[\\w\\d\\-\\_]+)*$"

// Vhost is a standalone, independent component of the WebRocket
// server which contains it's own users with permissions, channels,
// and other related settings.
type Vhost struct {
	path        string
	channels    map[string]*Channel
	ctx         *Context
	usersMtx    sync.Mutex
	chansMtx    sync.Mutex
	accessToken string
	permissions map[string]*Permission
}

// Creates new vhost for specified path. Path format allows you
// to use only letters, numbers, dashes and underscores. Also your
// path elemets can be separated with backslash.
func newVhost(ctx *Context, path string) (v *Vhost, err error) {
	re, err := regexp.Compile(vhostNamePattern)
	if !re.MatchString(path) {
		err = errors.New("Invalid path")
		return
	}
	v = &Vhost{path: path, ctx: ctx}
	v.GenerateAccessToken()
	v.channels = make(map[string]*Channel)
	return
}

// Generates access token for the backend connections. 
func (v *Vhost) GenerateAccessToken() string {
	var hash = sha1.New()
	var uuid = uuid.GenerateTime()
	hash.Write([]byte(uuid))
	v.accessToken = fmt.Sprintf("%x", hash.Sum([]byte{}))
	v.permissions = make(map[string]*Permission)
	return v.accessToken
}

// Generates single access token within this vhost.
func (v *Vhost) GenerateSingleAccessToken(pattern string) string {
	var p = NewPermission(pattern)
	v.permissions[p.Token()] = p
	return p.Token()
}

// Checks if specified token allows to access this vhost,
// and if so then returns associated permission.
func (v *Vhost) ValidateSingleAccessToken(token string) (p *Permission, ok bool) {
	p, ok = v.permissions[token]
	if ok {
		delete(v.permissions, token)
	}
	return
}

// Returns configured path of this vhost.
func (v *Vhost) Path() string {
	return v.path
}

// Opens new channel and registers it for this vhost.
func (v *Vhost) OpenChannel(name string) (ch *Channel, err error) {
	var exists bool

	_, exists = v.channels[name]
	if exists {
		err = errors.New(fmt.Sprintf("The '%s' channel already exists", name))
		return
	}
	ch, err = newChannel(name)
	if err != nil {
		return
	}
	v.chansMtx.Lock()
	defer v.chansMtx.Unlock()
	v.channels[name] = ch
	return
}

// Removes specified channel from this vhost.
func (v *Vhost) DeleteChannel(name string) (err error) {
	var ch *Channel

	ch, err = v.Channel(name)
	if err != nil {
		return
	}
	v.chansMtx.Lock()
	delete(v.channels, name)
	v.chansMtx.Unlock()
	ch.kill()
	// XXX: No idea what should we assume here? I think we should
	// assume that deleting the channel is only admin operation,
	// which shouldn't be done on live application (while channels
	// are automatically created when someone is subscribing it).
	// FIXME: We could send here unsubscribe message to all subscribers
	// to be sure that everything's fine.
	return
}

// Returns specified channel if exists.
func (v *Vhost) Channel(name string) (ch *Channel, err error) {
	var ok bool

	ch, ok = v.channels[name]
	if !ok {
		err = errors.New(fmt.Sprintf("The '%s' channel doesn't exist", name))
	}
	return
}

// Returns list of channels opened in this vhost.
func (v *Vhost) Channels() (channels []*Channel) {
	var i = 0
	var channel *Channel

	channels = make([]*Channel, len(v.channels))
	for _, channel = range v.channels {
		channels[i] = channel
		i += 1
	}
	return
}

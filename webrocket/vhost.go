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
	"crypto/rand"
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
	cmtx        sync.Mutex // channels' mutex
	tmtx        sync.Mutex // single access token mutex
	accessToken string
	permissions map[string]*Permission
}

// Creates new vhost for specified path. Path format allows you
// to use only letters, numbers, dashes and underscores. Also your
// path elemets can be separated with backslash.
func newVhost(ctx *Context, path string) (v *Vhost, err error) {
	re, err := regexp.Compile(vhostNamePattern)
	if !re.MatchString(path) {
		err = errors.New("invalid path")
		return
	}
	v = &Vhost{
		path:        path,
		ctx:         ctx,
		channels:    make(map[string]*Channel),
		permissions: make(map[string]*Permission),
	}
	return
}

// Generates access token for the backend connections. 
func (v *Vhost) GenerateAccessToken() string {
	var buf [32]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return ""
	}
	hash := sha1.New()
	hash.Write(buf[:])
	v.accessToken = fmt.Sprintf("%x", hash.Sum([]byte{}))
	if v.ctx != nil && v.ctx.storageEnabled() {
		v.ctx.storage.AddVhost(v.path, v.accessToken)
	}
	return v.accessToken
}

// AccessToken returns vhost's access token.
func (v *Vhost) AccessToken() string {
	return v.accessToken
}

// Generates single access token within this vhost.
func (v *Vhost) GenerateSingleAccessToken(pattern string) string {
	v.tmtx.Lock()
	defer v.tmtx.Unlock()
	var p = NewPermission(pattern)
	v.permissions[p.Token] = p
	return p.Token
}

// Checks if specified token allows to access this vhost,
// and if so then returns associated permission.
func (v *Vhost) ValidateSingleAccessToken(token string) (p *Permission, ok bool) {
	v.tmtx.Lock()
	defer v.tmtx.Unlock()
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
	v.cmtx.Lock()
	defer v.cmtx.Unlock()
	_, exists := v.channels[name]
	if exists {
		err = errors.New("channel already exists")
		return
	}
	ch, err = newChannel(name)
	if err != nil {
		return
	}
	if v.ctx != nil && v.ctx.storageEnabled() {
		err = v.ctx.storage.AddChannel(v.path, name)
		if err != nil {
			return
		}
	}
	v.channels[name] = ch
	return
}

// Removes specified channel from this vhost.
func (v *Vhost) DeleteChannel(name string) (ok bool) {
	v.cmtx.Lock()
	defer v.cmtx.Unlock()
	ch, ok := v.channels[name]
	if !ok {
		return
	}
	if v.ctx != nil && v.ctx.storageEnabled() {
		err := v.ctx.storage.DeleteChannel(v.path, name)
		if err != nil {
			return
		}
	}
	delete(v.channels, name)
	ch.Kill()
	return
}

// Returns specified channel if exists.
func (v *Vhost) Channel(name string) (ch *Channel, ok bool) {
	v.cmtx.Lock()
	defer v.cmtx.Unlock()
	ch, ok = v.channels[name]
	return
}

// Returns list of channels opened in this vhost.
func (v *Vhost) Channels() (channels []*Channel) {
	channels, i := make([]*Channel, len(v.channels)), 0
	for _, channel := range v.channels {
		channels[i] = channel
		i += 1
	}
	return
}

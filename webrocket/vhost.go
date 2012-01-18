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

// Pattern used to validate the vhost name.
var validVhostNamePattern = regexp.MustCompile("^/[\\w\\d\\-\\_]+(/[\\w\\d\\-\\_]+)*$")

// Vhost implements a standalone, independent component of the WebRocket
// server which contains it's own users with permissions, channels,
// and other related settings.
type Vhost struct {
	// A vhost's path.
	path string
	// A vhost's access token. 
	accessToken string
	// List of channels opened within the vhost. 
	channels map[string]*Channel
	// List of permissions generated for the vhost.
	permissions map[string]*Permission
	// Parent context.
	ctx *Context
	// Channel management semaphore
	cmtx sync.Mutex
	// Single access token's generation semaphore
	tmtx sync.Mutex
	// Internal semaphore
	imtx sync.Mutex
}

// Internal constructor
// -----------------------------------------------------------------------------

// Creates new vhost for specified path. Path format allows you to use only
// letters, numbers, dashes and underscores. Also your path elemets can be
// separated with backslash.
//
// Note: New vhost is not persisted until access token will be generated.
//
// ctx  - The parent context.
// path - The vhost's path name.
//
// Examples
//
//     v, err := newVhost(ctx, "/hello")
//
// Returns new vhost or error if something went wrong.
func newVhost(ctx *Context, path string) (v *Vhost, err error) {
	if !validVhostNamePattern.MatchString(path) {
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

// Exported
// -----------------------------------------------------------------------------

// GenerateAccessToken create a new access token for the backend connections.
// Threadsafe, called from main context and admin interface.
func (v *Vhost) GenerateAccessToken() string {
	v.imtx.Lock()
	defer v.imtx.Unlock()
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return ""
	}
	hash := sha1.New()
	hash.Write(buf[:])
	v.accessToken = fmt.Sprintf("%x", hash.Sum([]byte{}))
	if v.ctx != nil && v.ctx.isStorageEnabled() {
		// Write generated access token to the storage.
		v.ctx.storage.AddVhost(v.path, v.accessToken)
	}
	return v.accessToken
}

// AccessToken returns vhost's access token. Threadsafe, called from various
// backend connection's handlers.
func (v *Vhost) AccessToken() string {
	v.imtx.Lock()
	defer v.imtx.Unlock()
	return v.accessToken
}

// GenerateSingleAccessToken generates new single access token for the
// specified permissions. Threadsafe, called from various connection's
// handlers.
//
// pattern - The permission regexp to be attached to the token.
//
// Examples
//
//     token := v.GenerateSingleAccessToken("(foo|bar)")
//     println(token)
//     // => "f74fda...f54abd3"
//
func (v *Vhost) GenerateSingleAccessToken(pattern string) (token string) {
	if p, err := NewPermission(pattern); err == nil {
		token = p.Token()
		v.tmtx.Lock()
		defer v.tmtx.Unlock()
		v.permissions[token] = p
		return
	}
	return ""
}

// ValidateSingleAccessToken checks if the specified token allows to access
// this vhost, and if so then returns associated permission. Threadsafe,
// called from the various connection's handlers.
//
// token - The token to be checked.
//
// Returns related permission object and boolean status.  
func (v *Vhost) ValidateSingleAccessToken(token string) (p *Permission, ok bool) {
	v.tmtx.Lock()
	defer v.tmtx.Unlock()
	if p, ok = v.permissions[token]; ok {
		delete(v.permissions, token)
	}
	return
}

// Path returns configured path of this vhost.
func (v *Vhost) Path() string {
	return v.path
}

// OpenChannel creates new channel and registers it within the vhost.
// Threadsafe, may be called from the admin interface and affects other
// functions.
//
// name - The name of the new channel.
// kind - The type of the new channel.
//
// Examples
//
//     ch, _ = v.OpenChannel("hello")
//     println(ch.Name())
//     // => "hello"
//
// Returns new channel or error if something went wrong.
func (v *Vhost) OpenChannel(name string, kind ChannelType) (ch *Channel, err error) {
	v.cmtx.Lock()
	defer v.cmtx.Unlock()
	if _, ok := v.channels[name]; ok {
		err = errors.New("channel already exists")
		return
	}
	if ch, err = newChannel(name, kind); err != nil {
		return
	}
	if v.ctx != nil && v.ctx.isStorageEnabled() {
		// Write the channel info to the storage.
		if err = v.ctx.storage.AddChannel(v.path, name, kind); err != nil {
			return
		}
	}
	v.channels[name] = ch
	return
}

// DeleteChannel removes channel with the specified name from the vhost.
// Threadsafe, may be called from the admin interface and affect other functions.
//
// name - The name of the channel to be deleted.
//
// Returns whether this channel has been removed or not.
func (v *Vhost) DeleteChannel(name string) (ok bool) {
	v.cmtx.Lock()
	defer v.cmtx.Unlock()
	var ch *Channel
	if ch, ok = v.channels[name]; !ok {
		return
	}
	if v.ctx != nil && v.ctx.isStorageEnabled() {
		// Remove the channel from the storage.
		if err := v.ctx.storage.DeleteChannel(v.path, name); err != nil {
			return
		}
	}
	delete(v.channels, name)
	ch.Kill()
	return
}

// Channel returns specified channel if exists. Threadsafe, may be called
// from many places and being affected by other functions.
//
// name - The name of the channel to find.
//
// Returns a channel with given name and existance status.
func (v *Vhost) Channel(name string) (ch *Channel, err error) {
	v.cmtx.Lock()
	defer v.cmtx.Unlock()
	var ok bool
	if ch, ok = v.channels[name]; !ok {
		err = errors.New("channel doesn't exist")
	}
	return
}

// Channels returns list of the channels registered within the vhost.
func (v *Vhost) Channels() map[string]*Channel {
	v.cmtx.Lock()
	defer v.cmtx.Unlock()
	return v.channels
}

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
	"fmt"
	"log"
	"os"
)

// Context is a placeholder for general WebRocket's configuration
// and shared data. It's not possible to create any component
// without providing a context.
type Context struct {
	Ready     chan bool
	log       *log.Logger
	websocket *WebsocketEndpoint
	backend   *BackendEndpoint
	vhosts    map[string]*Vhost
}

// Creates new context.
func NewContext() (ctx *Context) {
	ctx = &Context{}
	ctx.log = log.New(os.Stderr, "", log.LstdFlags)
	ctx.vhosts = make(map[string]*Vhost)
	return ctx
}

// Returns configured logger.
func (ctx *Context) Log() *log.Logger {
	return ctx.log
}

// SetLog can be used to configure custom logger.
func (ctx *Context) SetLog(newLog *log.Logger) {
	ctx.log = newLog
}

// Registers new vhost under the specified path.
func (ctx *Context) AddVhost(path string) (v *Vhost, err error) {
	var exists bool

	_, exists = ctx.vhosts[path]
	if exists {
		err = errors.New(fmt.Sprintf("The '%s' vhost already exists", path))
		return
	}
	v, err = newVhost(ctx, path)
	if err == nil {
		ctx.vhosts[path] = v
		if ctx.websocket != nil {
			ctx.websocket.registerVhost(v)
		}
		if ctx.backend != nil {
			ctx.backend.registerVhost(v)
		}
	}
	return
}

// Removes and unregisters specified vhost.
func (ctx *Context) DeleteVhost(path string) (err error) {
	var vhost *Vhost

	vhost, err = ctx.Vhost(path)
	if err != nil {
		return
	}
	delete(ctx.vhosts, path)
	if ctx.websocket != nil {
		ctx.websocket.unregisterVhost(vhost)
	}
	if ctx.backend != nil {
		ctx.backend.unregisterVhost(vhost)
	}
	return
}

// Returns vhost from specified path if registered.
func (ctx *Context) Vhost(path string) (vhost *Vhost, err error) {
	var ok bool

	vhost, ok = ctx.vhosts[path]
	if !ok {
		err = errors.New(fmt.Sprintf("The '%s' vhost doesn't exist", path))
	}
	return
}

// Returns list of registered vhosts.
func (ctx *Context) Vhosts() (vhosts []*Vhost) {
	var i = 0
	var vhost *Vhost

	vhosts = make([]*Vhost, len(ctx.vhosts))
	for _, vhost = range ctx.vhosts {
		vhosts[i] = vhost
		i += 1
	}
	return
}

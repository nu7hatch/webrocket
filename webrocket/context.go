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
	"log"
	"os"
	"fmt"
)

// Context is a placeholder for all server configuration and
// shared data.
type Context struct {
	vhosts   map[string]*Vhost
	wsServ   *WebsocketServer
	mqServ   *MqServer
	Log      *log.Logger
}

// Creates new context.
func NewContext() *Context {
	ctx := new(Context)
	ctx.vhosts = make(map[string]*Vhost)
	ctx.Log = log.New(os.Stderr, "", log.LstdFlags)
	return ctx
}

// Registers new vhost within the context.
func (ctx *Context) AddVhost(path string) (*Vhost, error) {
	if ctx.wsServ == nil {
		return nil, errors.New("WebSockets server is not configured")
	}
	vhost := NewVhost(path)
	vhost.Log = ctx.Log
	ctx.vhosts[path] = vhost
	ctx.wsServ.Handler.(*ServeMux).AddHandler(path, vhost)
	ctx.Log.Printf("context: ADD_VHOST path='%s'", path)
	return vhost, nil
}

// Deletes specified vhost from the serve mux.
func (ctx *Context) DeleteVhost(path string) error {
	if ctx.wsServ == nil {
		return errors.New("WebSockets server is not configured")
	}
	vhost, ok := ctx.vhosts[path]
	if !ok {
		return errors.New(fmt.Sprintf("Vhost `%s` doesn't exist.", path))
	}
	ctx.wsServ.Handler.(*ServeMux).DeleteHandler(path)
	vhost.Stop()
	delete(ctx.vhosts, path)
	ctx.Log.Printf("context: DELETE_VHOST path='%s'", path)
	return nil
}

// Returns array with registered vhost paths.
func (ctx *Context) Vhosts() (vhosts []string) {
	vhosts, i := make([]string, len(ctx.vhosts)), 0
	for path := range ctx.vhosts {
		vhosts[i] = path
		i += 1
	}
	return vhosts
}

// Returns specified vhost.
func (ctx *Context) GetVhost(name string) (*Vhost, bool) {
	vhost, ok := ctx.vhosts[name]
	return vhost, ok
}
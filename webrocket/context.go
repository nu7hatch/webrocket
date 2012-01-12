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
	"log"
	"os"
	"sync"
	"path"
	"io"
	"fmt"
)

const CookieSize = 40

// Context is a placeholder for general WebRocket's configuration
// and shared data. It's not possible to create any component
// without providing a context.
type Context struct {
	Ready      chan bool
	log        *log.Logger
	websocket  *WebsocketEndpoint
	backend    *BackendEndpoint
	admin      *AdminEndpoint
	vhosts     map[string]*Vhost
	mtx        sync.Mutex
	storage    *storage
	storageOn  bool
	storageDir string
	cookie     string
}

// Creates new context.
func NewContext() *Context {
	return &Context{
		log:    log.New(os.Stderr, "", log.LstdFlags),
		vhosts: make(map[string]*Vhost),
	}
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
	ctx.mtx.Lock()
	defer ctx.mtx.Unlock()
	_, exists := ctx.vhosts[path]
	if exists {
		err = errors.New("vhost already exists")
		return
	}
	v, err = newVhost(ctx, path)
	if err != nil {
		return
	}
	v.GenerateAccessToken()
	// XXX: No need to save vhost to storage, it's done
	// internally by (*vhost).GenerateAccessToken().
	ctx.vhosts[path] = v
	if ctx.websocket != nil {
		ctx.websocket.registerVhost(v)
	}
	if ctx.backend != nil {
		ctx.backend.registerVhost(v)
	}
	return
}

// Removes and unregisters specified vhost.
func (ctx *Context) DeleteVhost(path string) (err error) {
	ctx.mtx.Lock()
	defer ctx.mtx.Unlock()
	vhost, ok := ctx.vhosts[path]
	if !ok {
		return errors.New("vhost doesn't exist")
	}
	if ctx.websocket != nil {
		ctx.websocket.unregisterVhost(vhost)
	}
	if ctx.backend != nil {
		ctx.backend.unregisterVhost(vhost)
	}
	if ctx.storageEnabled() {
		err = ctx.storage.DeleteVhost(path)
		if err != nil {
			return
		}
	}
	delete(ctx.vhosts, path)
	return
}

// Returns vhost from specified path if registered.
func (ctx *Context) Vhost(path string) (vhost *Vhost, err error) {
	ctx.mtx.Lock()
	defer ctx.mtx.Unlock()
	vhost, ok := ctx.vhosts[path]
	if !ok {
		err = errors.New("vhost doesn't exist")
	}
	return
}

// Returns list of registered vhosts.
func (ctx *Context) Vhosts() (vhosts []*Vhost) {
	vhosts, i := make([]*Vhost, len(ctx.vhosts)), 0
	for _, vhost := range ctx.vhosts {
		vhosts[i] = vhost
		i += 1
	}
	return
}

func (ctx *Context) storageEnabled() bool {
	// XXX: add different mutex
	return ctx.storage != nil && ctx.storageOn
}

func (ctx *Context) SetStorage(dir string) (err error) {
	ctx.storageOn = false
	ctx.storage, err = newStorage(dir)
	if err != nil {
		return
	}
	ctx.storageDir = dir
	vhosts, err := ctx.storage.Vhosts()
	if err != nil {
		return
	}
	for _, vstat := range vhosts {
		vhost, err := ctx.AddVhost(vstat.Name)
		if err != nil {
			continue // XXX: should i return in that case?
		}
		vhost.accessToken = vstat.AccessToken
		channels, err := ctx.storage.Channels(vstat.Name)
		if err != nil {
			continue // XXX: ...
		}
		for _, chstat := range channels {
			_, err := vhost.OpenChannel(chstat.Name)
			if err != nil {
				continue // XXX: ...
			}
		}
	}
	ctx.storageOn = true
	return
}

func (ctx *Context) Cookie() string {
	return ctx.cookie
}

func (ctx *Context) GenerateCookie(force bool) (err error) {
	if ctx.storageDir == "" {
		return errors.New("can't generate cookie, storage not set")
	}
	var buf = make([]byte, CookieSize)
	var cookieFile *os.File
	cookiePath := path.Join(ctx.storageDir, "cookie")
	if !force {
		cookieFile, err = os.Open(cookiePath)
		if err == nil {
			n, err := io.ReadFull(cookieFile, buf[:])
			if n == CookieSize && err == nil {
				ctx.cookie = string(buf[:])
				return nil
			}
		}
	}
	_, err = rand.Read(buf[:16])
	if err != nil {
		return
	}
	hash := sha1.New()
	hash.Write(buf[:16])
	ctx.cookie = fmt.Sprintf("%x", hash.Sum([]byte{}))
	cookieFile, err = os.Create(cookiePath)
	if err != nil {
		return
	}
	cookieFile.Write([]byte(ctx.cookie))
	cookieFile.Close()
	return
}

func (ctx *Context) Close() (err error) {
	// No need to lock, this func should be called only in one
	// place when exiting from the app.
	if ctx.storageEnabled() {
		err = ctx.storage.Save()
	}
	if ctx.websocket != nil {
		ctx.websocket.Kill()
	}
	if ctx.backend != nil { 
		ctx.backend.Kill()
	}
	return
}
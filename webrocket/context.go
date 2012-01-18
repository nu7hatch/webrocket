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
	"io"
	"log"
	"os"
	"path"
	"sync"
)

// The length of the cookie string.
const CookieSize = 40

// Context implements a placeholder for general WebRocket's configuration
// and shared data. It's not possible to create any of the components without
// providing a context. If context is dead, then everything else should be
// gracefully closed as well.
type Context struct {
	// Websocket endpoint interface.
	websocket *WebsocketEndpoint
	// Backend endpoint interface.
	backend *BackendEndpoint
	// Admin endpoint interface.
	admin *AdminEndpoint
	// List of registered vhosts.
	vhosts map[string]*Vhost
	// The persistent storage.
	storage *storage
	// Storage status.
	storageOn bool
	// Path to the storage directory.
	storageDir string
	// The node admin's cookie.
	cookie string
	// Internal logger.
	log *log.Logger
	// Internal semaphore.
	mtx sync.Mutex
}

// Exported constructor
// -----------------------------------------------------------------------------

// NewContext creates and preconfigures new context.
//
// Returns a new context.
func NewContext() *Context {
	return &Context{
		log:    log.New(os.Stderr, "", log.LstdFlags),
		vhosts: make(map[string]*Vhost),
	}
}

// Internal
// -----------------------------------------------------------------------------

// isStorageEnabled returns whether the internal storage can be accessed to
// read/write or not. Not threadsafe, no need while it's called only from
// within the synchronized context's and vhost's functions. 
func (ctx *Context) isStorageEnabled() bool {
	return ctx.storage != nil && ctx.storageOn
}

// Exported
// -----------------------------------------------------------------------------

// Log returns a configured logger.
func (ctx *Context) Log() *log.Logger {
	return ctx.log
}

// SetLog configures custom logger.
//
// newLog - The logger instance to be assigned with the context.
//
// Examples
//
//     logger := log.NewLogger(os.Stderr, "My Logger!", log.LstdFlags)
//     ctx.SetLog(logger)
//
func (ctx *Context) SetLog(newLog *log.Logger) {
	ctx.log = newLog
}

// Cookie returns the value of the node admin's cookie hash.
func (ctx *Context) Cookie() string {
	return ctx.cookie
}

// GenerateCookie generates new random node admin's cookie hash and saves
// it to the storage dir in the 'cookie' file. If no force flag specified
// then cookie will be not overwritten during the further calls of this
// function.
//
// force - If true, then generates new cookie and overwrites existing one.
//
// Returns an error if something went wrong.
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
	// Generate new cookie if there's none or the force flag is enabled.
	if _, err = rand.Read(buf[:16]); err != nil {
		return
	}
	hash := sha1.New()
	hash.Write(buf[:16])
	ctx.cookie = fmt.Sprintf("%x", hash.Sum([]byte{}))
	if cookieFile, err = os.Create(cookiePath); err != nil {
		return
	}
	cookieFile.Write([]byte(ctx.cookie))
	cookieFile.Close()
	return
}

// SetStorage sets the storage directory and preloads existing information
// about the vhosts, channels, etc. Storage is automatically marked as
// available if everything's find at return. Not threadsafe, storage directory
// shall be set only once from the main goroutine. It's not possible to
// change storage dir while executing the
// program.
//
// dir - A path to the storage directory.
//
// Returns an error if something went wrong.
func (ctx *Context) SetStorage(dir string) (err error) {
	ctx.storageOn = false
	if ctx.storage, err = newStorage(dir); err != nil {
		return
	}
	ctx.storageDir = dir
	// Loading the list persisted vhosts. 
	vhosts, err := ctx.storage.Vhosts()
	if err != nil {
		return err
	}
	for _, vstat := range vhosts {
		vhost, err := ctx.AddVhost(vstat.Name)
		if err != nil {
			return err
		}
		vhost.accessToken = vstat.AccessToken
		// Loading all the channels registered within this vhost.
		channels, err := ctx.storage.Channels(vstat.Name)
		if err != nil {
			return err
		}
		for _, chstat := range channels {
			_, err := vhost.OpenChannel(chstat.Name, chstat.Kind)
			if err != nil {
				return err
			}
		}
	}
	// Everything's fine, enabling the access to the storage.
	ctx.storageOn = true
	return
}

// AddVhost registers new vhost under the specified path. Threadsafe,
// may be called from the admin endpoint or storage loader and its
// execution affects Vhost, Vhosts and DeleteVhost functions.
//
// path - The path to register the vhost under.
//
// Returns new vhost or an error if something went wrong.
func (ctx *Context) AddVhost(path string) (v *Vhost, err error) {
	ctx.mtx.Lock()
	defer ctx.mtx.Unlock()
	if _, ok := ctx.vhosts[path]; ok {
		err = errors.New("vhost already exists")
		return
	}
	if v, err = newVhost(ctx, path); err != nil {
		return
	}
	// XXX: GenerateAccessToken internally adds this vhost to the storage,
	// so we don't need to save it manually here.
	v.GenerateAccessToken()
	ctx.vhosts[path] = v
	// Registering the vhost under the websocket and backend endpoints.
	if ctx.websocket != nil {
		ctx.websocket.registerVhost(v)
	}
	if ctx.backend != nil {
		ctx.backend.registerVhost(v)
	}
	return
}

// DeleteVhost removes and unregisters vhost from the specified path.
// Threadsafe, may be called from the admin endpoint or storage loader and
// its execution affects Vhost, Vhosts and AddVhost functions.
//
// path - The path to the vhost to be deleted.
//
// Returns an error if something went wrong.
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
	if ctx.isStorageEnabled() {
		// Remove the vhost entry from the storage.
		if err = ctx.storage.DeleteVhost(path); err != nil {
			return
		}
	}
	delete(ctx.vhosts, path)
	return
}

// Vhost finds a vhost registered under the specified path and returns
// it if exists. Threadsafe, may be called from the admin endpoint or
// storage loader and its execution affects Vhosts, DeleteVhost and
// AddVhost functions.
//
// path - The vhost's path to be found.
//
// Returns vhost from specified path or an error if something went wrong
// or vhost doesn't exist.
func (ctx *Context) Vhost(path string) (vhost *Vhost, err error) {
	ctx.mtx.Lock()
	defer ctx.mtx.Unlock()
	var ok bool
	if vhost, ok = ctx.vhosts[path]; !ok {
		err = errors.New("vhost doesn't exist")
	}
	return
}

// Vhosts returns list of all the registered vhosts. Threadsafe, may be
// called from the admin endpoint or storage loader and its execution
// affects Vhost, DeleteVhost and AddVhost functions.
func (ctx *Context) Vhosts() map[string]*Vhost {
	ctx.mtx.Lock()
	defer ctx.mtx.Unlock()
	return ctx.vhosts
}

// Kill closes the context and cleans up all internal endpoints, closes
// all connections and in general makes everyone happy. Not threadsafe,
// no need to lock while this func shall be called only in one place - when
// exiting from the application.
//
// Returns an error if something went wrong.
func (ctx *Context) Kill() (err error) {
	if ctx.isStorageEnabled() {
		// Make sure that all the data are saved.
		err = ctx.storage.Save()
		ctx.storage.Kill()
	}
	// Kill all endpoints.
	if ctx.websocket != nil {
		ctx.websocket.Kill()
	}
	if ctx.backend != nil {
		ctx.backend.Kill()
	}
	if ctx.admin != nil {
		ctx.admin.Kill()
	}
	return
}

// NewWebsocketEndpoint creates a new websocket endpoint, registers handlers
// for all the existing vhosts and registers it within the context.
//
// addr - The host and port to which this endpoint will be bound.
//
// Examples
//
//     e := ctx.NewWebsocketEndpoint(":8080")
//     if err := e.ListenAndServe(); err != nil {
//         println(err.Error())
//     }
//
// Returns a configured endpoint. 
func (ctx *Context) NewWebsocketEndpoint(addr string) Endpoint {
	we := newWebsocketEndpoint(ctx, addr)
	for _, vhost := range ctx.vhosts {
		we.registerVhost(vhost)
	}
	ctx.websocket = we
	return we
}

// NewBackendEndpoint creates a new backend endpoint, registers handlers
// for all the existing vhosts and registers it within the context.
//
// addr - The host and port to which this endpoint will be bound.
//
// Examples
//
//     e := ctx.NewBackendEndpoint(":8080")
//     if err := e.ListenAndServe(); err != nil {
//         println(err.Error())
//     }
//
// Returns a configured endpoint.
func (ctx *Context) NewBackendEndpoint(addr string) Endpoint {
	be := newBackendEndpoint(ctx, addr)
	for _, vhost := range ctx.vhosts {
		be.registerVhost(vhost)
	}
	ctx.backend = be
	return be
}

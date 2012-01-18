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
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"sync"
)

// WebsocketEndpoint implements a wrapper for the websockets server
// instance. For now, it extends the standard http.Server functionality
// with WebRocket specific stuff.
type WebsocketEndpoint struct {
	*http.Server

	// Context to which the endpoint belongs.
	ctx *Context
	// Information whether an endpoint is alive or not.
	alive bool
	// List of registered handlers (handler per vhost).
	handlers *WebsocketServeMux
	// Internal semaphore.
	mtx sync.Mutex
	// Internal Logger.
	log *log.Logger
}

// Internal constructors
// -----------------------------------------------------------------------------

// netWebsocketEndpoint creates new websockets endpoint object configured to
// be bound to specified address. If no host specified in the address
// (eg. `:8080`). then will be bound to all available interfaces.
//
// `ctx`  - The parent context.
// `addr` - The host and port to which this endpoint will be bound.
//
// Returns new configured websocket endpoint.
func newWebsocketEndpoint(ctx *Context, addr string) *WebsocketEndpoint {
	mux := NewWebsocketServeMux()
	return &WebsocketEndpoint{
		handlers: mux,
		Server:   &http.Server{Addr: addr, Handler: mux},
		ctx:      ctx,
		log:      ctx.log,
	}
}

// Internal
// -----------------------------------------------------------------------------

// registerVhost registers a new handler for the specified vhost. Threadsafe
// thanks to using a threadsafe serve mux.
//
// vhost - The vhost to be registered.
//
func (w *WebsocketEndpoint) registerVhost(vhost *Vhost) {
	h := newWebsocketHandler(vhost, w)
	w.handlers.AddHandler(vhost.Path(), h)
}

// unregisterVhost removes a handler for the specified vhost if such has
// been registered before. Threadsafe thanks to using a threadsafe serve mux.
//
// vhost - The vhost to be removed.
//
func (w *WebsocketEndpoint) unregisterVhost(vhost *Vhost) {
	w.handlers.DeleteHandler(vhost.Path())
}

// Exported
// -----------------------------------------------------------------------------

// Addr returns an address to which this endpoint is bound.
func (w *WebsocketEndpoint) Addr() string {
	return w.Server.Addr
}

// ListenAndServe listens on the TCP network address addr and then calls
// Serve with handler to handle requests on incoming connections.
//
// Returns an error if something went wrong.
func (w *WebsocketEndpoint) ListenAndServe() error {
	addr := w.Server.Addr
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	w.alive = true
	return w.Server.Serve(l)
}

// ListenAndServeTLS acts identically to ListenAndServe, except that it expects
// HTTPS connections. Additionally, files containing a certificate and matching
// private key for the server must be provided. If the certificate is signed by
// a certificate authority, the certFile should be the concatenation of the
// server's certificate followed by the CA's certificate.

// One can use generate_cert.go in crypto/tls to generate cert.pem and key.pem.
//
// certFile - Path to the TLS certificate file.
// certKey  - Path to the certificate's private key.
//
// Returns an error if something went wrong.
func (w *WebsocketEndpoint) ListenAndServeTLS(certFile, certKey string) (err error) {
	addr := w.Server.Addr
	config := &tls.Config{Rand: rand.Reader, NextProtos: []string{"http/1.1"}}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, certKey)
	if err != nil {
		return
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}
	tlsListener := tls.NewListener(l, config)
	w.alive = true
	return w.Server.Serve(tlsListener)
}

// IsAlive returns whether the endpoint is alive or not.
func (w *WebsocketEndpoint) IsAlive() bool {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.alive
}

// Kill stops all registered vhost handlers and marks this endpoint as dead.
func (w *WebsocketEndpoint) Kill() {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	if w.alive {
		w.alive = false
		w.handlers.KillAll()
	}
}

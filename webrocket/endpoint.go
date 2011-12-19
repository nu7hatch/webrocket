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

import "sync"

// Endpoint is an interface representing all endpoints installed
// on the context's rack.
type Endpoint interface {
	ListenAndServe() error
	ListenAndServeTLS(certFile, certKey string) error
	Addr() string
	IsRunning() bool
}

// Base structure for all endpoints.
type BaseEndpoint struct {
	ctx       *Context
	isRunning bool
	mtx       sync.Mutex
}

// Terminates endpoint execution.
func (e *BaseEndpoint) kill() {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	e.isRunning = false
}

// Marks this endpoint as running.
func (e *BaseEndpoint) alive() {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	e.isRunning = true
}

// Returns true if this endpoint is activated.
func (e *BaseEndpoint) IsRunning() bool {
	return e.isRunning
}
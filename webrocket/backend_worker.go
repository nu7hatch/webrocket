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
	"encoding/json"
	"io"
	"sync"
	"time"
)

// Backend worker defaults.
const (
	backendWorkerHeartbeatInterval = 2 * time.Second
	backendWorkerHeartbeatLiveness = 3
	backendWorkerHeartbeatExpiry   = backendWorkerHeartbeatInterval * backendWorkerHeartbeatLiveness
)

// BackendWorker is a wrapper for the backend worker's connection
// with majordomo's keep alive mechanism.
type BackendWorker struct {
	// Unique identifier of the worker.
	id string
	// The underlaying connection.
	conn *backendConnection
	// The expiration time.
	expiry time.Time
	// The heartbeat scheduled time.
	heartbeatAt time.Time
	// Internal semaphore.
	mtx sync.Mutex
}

// Internal constructor
// -----------------------------------------------------------------------------

// newBackendWorker creates a new backend worker object. To make the worker
// running the 'listen' function needs to be executed in a separate goroutine.
//
// conn - The backend connection to be wrapped.
// id   - The worker's unique identifier.
//
func newBackendWorker(conn *backendConnection, id string) (a *BackendWorker) {
	a = &BackendWorker{
		id:          id,
		conn:        conn,
		expiry:      time.Now(),
		heartbeatAt: time.Now().Add(backendWorkerHeartbeatInterval),
	}
	a.updateExpiration()
	return a
}

// Internal
// -----------------------------------------------------------------------------

// updateExpiration refreshes the expiration date which makes the worker
// alive until then.
func (a *BackendWorker) updateExpiration() {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.expiry = time.Now().Add(backendWorkerHeartbeatExpiry)
}

// listen implements an event loop which keeps the worker's connection alive
// by exchanging the hearbeats.
func (a *BackendWorker) listen() {
	defer a.Kill()
	a.conn.SetTimeout(int64(backendWorkerHeartbeatInterval))
	for {
		if !a.IsAlive() {
			break
		}
		req, err := a.conn.Recv()
		if err != nil && err == io.EOF {
			// End of file reached...
			break
		}
		if req != nil {
			switch req.Command {
			case "HB": // Heartbeat
				a.updateExpiration()
			case "QT": // Quit
				break
			}
		}
		if a.heartbeatAt.Before(time.Now()) {
			// Send the heartbeat message and update schedule if it's time.
			a.conn.Send("HB")
			a.heartbeatAt = time.Now().Add(backendWorkerHeartbeatInterval)
		}
	}
}

// Exported
// -----------------------------------------------------------------------------

// Id returns the worker's unique identifier.
func (a *BackendWorker) Id() string {
	return a.id
}

// Trigger sends given payload directly to the worker.
//
// payload - The data to be send
//
func (a *BackendWorker) Trigger(payload interface{}) (err error) {
	if a.IsAlive() {
		if frame, err := json.Marshal(payload); err == nil {
			err = a.conn.Send("TR", string(frame))
		}
	}
	return
}

// IsAlive returns whether this worker is working or not. Threadsafe, may be
// called from the various handlers.
func (a *BackendWorker) IsAlive() bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.expiry.After(time.Now()) && a.conn.IsAlive()
}

// Kill terminates execution of this worker and closes its underlaying
// connection. Threadsafe, may be called from the many handlers or affect
// other functions.
func (a *BackendWorker) Kill() {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	if a.conn != nil {
		a.conn.Send("QT")
		a.conn.Kill()
		a.expiry = time.Now()
	}
}

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
	"time"
)

const (
	backendAgentHeartbeatInterval = 2 * time.Second
	backendAgentHeartbeatLiveness = 3
	backendAgentHeartbeatExpiry   = backendAgentHeartbeatInterval + backendAgentHeartbeatLiveness
)

// BackendAgent represents single Backend Application client connection.
type BackendAgent struct {
	*connection
	id       []byte
	endpoint *BackendEndpoint
	expiry   time.Time
}

// newBackendAgent creates reference to specified backend application's client
// connection. Each client uses separate goroutine to deal with the
// outgoing messages.
func newBackendAgent(endpoint *BackendEndpoint, v *Vhost, id []byte) (a *BackendAgent) {
	a = &BackendAgent{
		id:         id,
		endpoint:   endpoint,
		connection: newConnection(v),
		expiry:     time.Now(),
	}
	a.updateExpiration()
	go a.heartbeat()
	return a
}

// Returns identifier of this agent.
func (a *BackendAgent) Id() string {
	return string(a.id)
}

// Sends given payload to the client.
func (a *BackendAgent) Trigger(payload interface{}) (err error) {
	if a.IsAlive() {
		var frame []byte
		frame, err = json.Marshal(payload)
		if err != nil {
			return
		}
		err = a.endpoint.SendTo(a.id, true, "TR", string(frame))
	}
	return
}

// Returns true if this agent is alive.
func (a *BackendAgent) IsAlive() bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.expiry.After(time.Now())
}

// Turns off the agent's alive state.
func (a *BackendAgent) Kill() {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.endpoint.SendTo(a.id, true, "QT")
	a.expiry = time.Now()
}

func (a *BackendAgent) updateExpiration() {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.expiry = a.expiry.Add(backendAgentHeartbeatExpiry)
}

func (a *BackendAgent) heartbeat() {
	for {
		if !a.IsAlive() {
			break
		}
		<-time.After(backendAgentHeartbeatInterval)
		a.endpoint.SendTo(a.id, true, "HB")
	}
}
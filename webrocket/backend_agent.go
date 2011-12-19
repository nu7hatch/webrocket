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

// BackendAgent represents single Backend Application client connection.
type BackendAgent struct {
	*connection
	id       []byte
	endpoint *BackendEndpoint
}

// newBackendAgent creates reference to specified backend application's client
// connection. Each client uses separate goroutine to deal with the
// outgoing messages.
func newBackendAgent(endpoint *BackendEndpoint, v *Vhost, id []byte) (a *BackendAgent) {
	a = &BackendAgent{id: id}
	a.connection = newConnection(v)
	a.endpoint = endpoint
	return
}

// Returns identifier of this agent.
func (a *BackendAgent) Id() string {
	return string(a.id)
}

// Sends given payload to the client.
func (a *BackendAgent) Send(payload interface{}) {
	if a.IsAlive() {
		// XXX: need to check if this mutex here is really needed...
		a.mtx.Lock()
		defer a.mtx.Unlock()
		a.endpoint.SendTo(a.id, payload)
	}
}
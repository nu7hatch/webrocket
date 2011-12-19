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

import "encoding/json"

type backendAPI struct {
	endpoint *backendEndpoint
}

func (b *BackendEndpoint) newBackendAPI() *backendAPI {
	return &backendAPI{endpoint: b}
}

func (api *backendAPI) dispatch(id, payload []byte) {
	msg, err := newMessageFromJSON(payload)
	if err != nil {
		// invalid message received
		return
	}
	switch msg.Event() {
	case "auth":
		api.handleAuth(msg)
	case "broadcast":
		api.handleBroadcast(msg)
	case "direct":
		api.handleDirect(msg)
	case "openChannel":
		api.handleOpenChannel(msg)
	case "pong":
		api.handlePong(msg)
	default:
		api.notFound(msg)
	}
}

func (api *backendAPI) handleAuth(msg *Message) {
	
}

func (api *backendAPI) handleBroadcast(msg *Message) {
	
}

func (api *backendAPI) handleDirect(msg *Message) {
	
}

func (api *backendAPI) handleOpenChannel(msg *Message) {
	
}

func (api *backendAPI) handlePong(msg *Message) {
	
}

func (api *backendAPI) notFound(msg *Message) {
	
}
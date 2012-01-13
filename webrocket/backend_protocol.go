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
	"fmt"
	"encoding/json"
)

var backendReqProtocol = map[string]func(*backendRequest)(string,int){
	"BC": backendReqHandleBroadcast,
	"OC": backendReqHandleOpenChannel,
	"CC": backendReqHandleCloseChannel,
	"AT": backendReqHandleSingleAccessTokenRequest,
}

// Helper for logging backend handler's statuses.
func backendStatusLog(b *BackendEndpoint, v *Vhost, status string, code int, msg string) {
	path := "..."
	if v != nil {
		path = v.Path()
	}
	b.log.Printf("backend[%s]: %d %s; %s", path, code, status, msg)
}

// Helper for logging protocol errors and and seding it to
// the client.
func backendError(b *BackendEndpoint, v *Vhost, aid []byte, error string,
	code int, msg string) {
	b.SendTo(aid, true, "ER", fmt.Sprintf("%d", code))
	backendStatusLog(b, v, error, code, msg)
}

// backendDealerDispatch takes a message received from the agent
// and handles it in appropriate way accoring to the backend worker
// protocol specification.
func backendDealerDispatch(r *backendRequest) (status string, code int) {
	var agent *BackendAgent
	lobby, ok := r.endpoint.lobbys[r.vhost.Path()]
	if !ok {
		// Something's fucked up, it should never happen
		status, code = "Internal error", 500
		return
	}
	switch r.cmd {
	case "QT":
		agent, ok = lobby.getAgentById(string(r.id))
		if ok {
			lobby.deleteAgent(agent);
			status, code = "Disconnected", 309
			return
		}
		status, code = "Forbidden", 403
		return
	case "RD":
		// First message from the agent, means it's ready to work
		agent = newBackendAgent(r.endpoint, r.vhost, r.id)
		lobby.addAgent(agent)
		status, code = "Ready", 300
	case "HB":
		agent, ok = lobby.getAgentById(string(r.id))
		if ok {
			// Just update expiration time...
			agent.updateExpiration()
			status, code = "Heartbeat", 301
		} else {
			// Seems that agent sent heartbeat after liveness period,
			// we have to send a quit message restart it.
			r.Reply("QT")
			status, code = "Expired", 408
		}
	default:
		status, code = "Bad request", 400
	}
	return 
}

// backendReqDispatch takes an incoming message and handles it
// in appropriate way accoring to the backend worker protocol
// specification.
func backendReqDispatch(r *backendRequest) (string, int) {
	handlerFunc, ok := backendReqProtocol[r.cmd]
	if !ok {
		return "Bad request", 400
	}
	return handlerFunc(r)
}

// The 'BC' (broadcast) event handler.
func backendReqHandleBroadcast(r *backendRequest) (string, int) {
	// Getting data from payload...
	if len(r.msg) < 3 {
		return "Bad request", 400
	}
	chanName, eventName := string(r.msg[0]), string(r.msg[1])
	if chanName == "" || eventName == "" {
		return "Bad request", 400
	}
	rawData := r.msg[2]
	var data map[string]interface{}
	err := json.Unmarshal(rawData, &data)
	if err != nil {
		data = map[string]interface{}{}
	}
	// Checking if channel exists...
	channel, ok := r.vhost.Channel(chanName)
	if !ok || channel == nil {
		return "Channel not found", 454
	}
	// Extending data with sender id and channel information before
	// pass it forward. Finally, broadcasting and replying to the client.
	data["channel"] = chanName
	channel.Broadcast(&map[string]interface{}{eventName: data})
	r.Reply("OK")
	return "Broadcasted", 204
}

// The 'OC' (open channel) event handler.
func backendReqHandleOpenChannel(r *backendRequest) (string, int) {
	// Getting data from payload...
	if len(r.msg) < 1 {
		return "Bad request", 400
	}
	chanName := string(r.msg[0])
	if chanName == "" {
		return "Bad request", 400
	}
	// Checking if channel already exists
	_, ok := r.vhost.Channel(chanName)
	if ok {
		r.Reply("OK")
		return "Channel exists", 251
	}
	// Trying to create if not exists...
	_, err := r.vhost.OpenChannel(chanName)
	if err != nil {
		return "Invalid channel name", 451
	}
	// Channel created, sending success response
	r.Reply("OK")
	return "Channel opened", 250
}

// The 'CC' (close channel) event handler.
func backendReqHandleCloseChannel(r *backendRequest) (string, int) {
	// Getting data from payload...
	if len(r.msg) < 1 {
		return "Bad request", 400
	}
	chanName := string(r.msg[0])
	if chanName == "" {
		return "Bad request", 400
	}
	// Deleting channel if exists
	ok := r.vhost.DeleteChannel(chanName)
	if !ok {
		return "Channel not found", 454
	}
	// Channel deleted, sending success response
	r.Reply("OK")
	return "Channel closed", 252
}

// The 'AT' (access token) event handler.
func backendReqHandleSingleAccessTokenRequest(r *backendRequest) (string, int) {
	// Getting data from payload...
	pattern := ".*"
	if len(r.msg) > 0 {
		pattern = string(r.msg[0])
	}
	// Generating a single access token for specified permissions...
	token := r.vhost.GenerateSingleAccessToken(pattern)
	// ... and sending it in the response
	r.Reply("AT", token)
	return "Single access token generated", 270 
}
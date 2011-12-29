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

// The '__singleAccessToken' event's payload.
func backendEventSingleAccessToken(token string) map[string]interface{} {
	return map[string]interface{}{
		"__singleAccessToken": map[string]interface{}{
			"token": token,
		},
	}
}

// It's just plain struct to gather all event handlers together.
// The idea is to have one global copy of the protocol object
// and share it across all connected agents.
type backendReqProtocol struct{}

// dispatch takes a message incoming from the REQ client and
// handles it in appropriate way according to the protocol
// specification.
func (p *backendReqProtocol) dispatch(b *BackendEndpoint,
	vhost *Vhost, aid []byte, msg *Message) {
	switch msg.Event() {
	case "broadcast":
		p.handleBroadcast(b, vhost, aid, msg)
	case "openChannel":
		p.handleOpenChannel(b, vhost, aid, msg)
	case "closeChannel":
		p.handleCloseChannel(b, vhost, aid, msg)
	case "singleAccessToken":
		p.handleSingleAccessToken(b, vhost, aid, msg)
	default:
		p.notFound(b, vhost, aid, msg)
	}
}

// Shorthand for logging operations.
func (p *backendReqProtocol) log(b *BackendEndpoint, vhost *Vhost,
	aid []byte, code string, a ...interface{}) {
	msg, ok := logMsg[code]
	if ok {
		path := "..."
		if vhost != nil {
			path = vhost.Path()
		}
		a = append([]interface{}{"req", path, code}, a...)
		b.log.Printf(msg, a...)
	}
}

// Shorthand for handling errors.
func (p *backendReqProtocol) error(b *BackendEndpoint, vhost *Vhost,
	aid []byte, code string, err map[string]interface{}, a ...interface{}) {
	b.SendTo(aid, err)
	p.log(b, vhost, aid, code, a...)
}

// The 'broadcast' event handler.
func (p *backendReqProtocol) handleBroadcast(b *BackendEndpoint,
	vhost *Vhost, aid []byte, msg *Message) {
	// Getting data from payload...
	chanName, _ := msg.Get("channel").(string)
	eventName, _ := msg.Get("event").(string)

	// Checking if message's payload is valid 
	if chanName == "" || eventName == "" {
		// Bad request
		p.error(b, vhost, aid, "400", errorBadRequest, "Invalid payload params")
		return
	}
	// Data field is optional, so when it's empty we have to
	// assign an empty map.
	data, ok := msg.Get("data").(map[string]interface{})
	if !ok {
		data = map[string]interface{}{}
	}
	// Checking if channel exists...
	channel, err := vhost.Channel(chanName)
	if err != nil || channel == nil {
		// Channel not found
		p.error(b, vhost, aid, "454", errorChannelNotFound, chanName)
		return
	}
	// Extending data with sender id and channel information before
	// pass it forward.
	data["channel"] = chanName
	// ... and finally broadcasting it on the channel.
	channel.Broadcast(&map[string]interface{}{eventName: data})
	b.SendTo(aid, okBroadcasted)
	p.log(b, vhost, aid, "204", eventName, chanName, "...")
}

// The 'openChannel' event handler.
func (p *backendReqProtocol) handleOpenChannel(b *BackendEndpoint,
	vhost *Vhost, aid []byte, msg *Message) {
	// Getting data from payload... 
	chanName, ok := msg.Get("channel").(string)
	if !ok || chanName == "" {
		// Bad request
		p.error(b, vhost, aid, "400", errorBadRequest, "Invalid payload params")
		return
	}
	// Checking if channel already exists
	_, err := vhost.Channel(chanName)
	if err == nil {
		// If channel exists, then return success response
		b.SendTo(aid, okChannelExists)
		p.log(b, vhost, aid, "251", chanName)
		return
	}
	// Trying to create if not exists...
	_, err = vhost.OpenChannel(chanName)
	if err != nil {
		// Invalid channel name
		p.error(b, vhost, aid, "451", errorInvalidChannelName, chanName)
		return
	}
	// Channel created, sending success response
	b.SendTo(aid, okChannelOpened)
	p.log(b, vhost, aid, "250", chanName)
}

// The 'closeChannel' event handler.
func (p *backendReqProtocol) handleCloseChannel(b *BackendEndpoint,
	vhost *Vhost, aid []byte, msg *Message) {
	// Getting data from payload... 
	chanName, ok := msg.Get("channel").(string)
	if !ok || chanName == "" {
		// Bad request
		p.error(b, vhost, aid, "400", errorBadRequest, "Invalid payload params")
		return
	}
	// Deleting channel if exists
	err := vhost.DeleteChannel(chanName)
	if err != nil {
		// Channel not found
		p.error(b, vhost, aid, "454", errorChannelNotFound, chanName)
		return
	}
	// Channel deleted, sending success response
	b.SendTo(aid, okChannelClosed)
	p.log(b, vhost, aid, "252", chanName)
}

// The 'singleAccessToken' event handler.
func (p *backendReqProtocol) handleSingleAccessToken(b *BackendEndpoint,
	vhost *Vhost, aid []byte, msg *Message) {
	// Getting data from payload... 
	pattern, ok := msg.Get("permission").(string)
	if !ok || pattern == "" {
		// Set default pattern if not present...
		pattern = ".*"
	}
	// Generating a single access token for specified permissions...
	token := vhost.GenerateSingleAccessToken(pattern)
	// ... and sending it in the response
	b.SendTo(aid, backendEventSingleAccessToken(token))
	p.log(b, vhost, aid, "253", pattern)
}

// Handles situation when requested event is not supported by
// the WebRocket Frontend Protocol.
func (p *backendReqProtocol) notFound(b *BackendEndpoint,
	vhost *Vhost, aid []byte, msg *Message) {
	p.error(b, vhost, aid, "400", errorBadRequest, "Event not implemented")
}

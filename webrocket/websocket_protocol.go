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

// The '__connected' event's payload.
func websocketEventConnected(sid string) map[string]interface{} {
	return map[string]interface{}{
		"__connected": map[string]interface{}{
			"sid": sid,
		},
	}
}

// The '__authenticated' event's payload.
func websocketEventAuthenticated() map[string]interface{} {
	return map[string]interface{}{
		"__authenticated": map[string]interface{}{},
	}
}

// The '__subscribed' event's payload.
func websocketEventSubscribed(chanName string) map[string]interface{} {
	return map[string]interface{}{
		"__subscribed": map[string]interface{}{
			"channel": chanName,
		},
	}
}

// The '__unsubscribed' event's payload.
func websocketEventUnsubscribed(chanName string) map[string]interface{} {
	return map[string]interface{}{
		"__unsubscribed": map[string]interface{}{
			"channel": chanName,
		},
	}
}

// The '__closed' event's payload.
func websocketEventClosed(sid string) map[string]interface{} {
	return map[string]interface{}{
		"__closed": map[string]interface{}{
			"sid": sid,
		},
	}
}

// It's just plain struct to gather all event handlers together.
// The idea is to have one global copy of the protocol object
// and share it across all connected clients.
type websocketProtocol struct{}

// dispatch takes an incoming message and handles it in appropriate
// way according to the protocol specification.
func (p *websocketProtocol) dispatch(c *WebsocketClient, msg *Message) bool {
	switch msg.Event() {
	case "broadcast":
		p.handleBroadcast(c, msg)
	case "trigger":
		p.handleTrigger(c, msg)
	case "subscribe":
		p.handleSubscribe(c, msg)
	case "unsubscribe":
		p.handleUnsubscribe(c, msg)
	case "auth":
		p.handleAuth(c, msg)
	case "pong":
		p.handlePong(c, msg)
	case "close":
		p.handleClose(c, msg)
		return false
	default:
		p.notFound(c, msg)
	}
	return true
}

// Shorthand for logging operations.
func (p *websocketProtocol) log(c *WebsocketClient, code string, a ...interface{}) {
	msg, ok := logMsg[code]
	if ok {
		a = append([]interface{}{"websocket", c.vhost.Path(), code}, a...)
		c.log.Printf(msg, a...)
	}
}

// Shorthand for handling errors.
func (p *websocketProtocol) error(c *WebsocketClient, code string,
	err map[string]interface{}, a ...interface{}) {
	c.Send(err)
	p.log(c, code, a...)
}

// The 'auth' event handler.
func (p *websocketProtocol) handleAuth(c *WebsocketClient, msg *Message) {
	// Getting data from payload...
	token, ok := msg.Get("token").(string)
	if !ok || token == "" {
		// Bad request
		p.error(c, "400", errorBadRequest, "Invalid payload params")
		return
	}
	// Closing current session if authenticated...
	if c.IsAuthenticated() {
		c.clearSubscriptions()
		c.authenticate(nil) // ... and terminate this session
	}
	// Checking if given single access token exists
	perm, ok := c.vhost.ValidateSingleAccessToken(token)
	if !ok || perm == nil {
		// Unauthorized
		p.error(c, "402", errorUnauthorized, "Invalid access token")
		return
	}
	// And if everything's fine then authenticate the client's
	// session and send a confirmation message. 
	c.authenticate(perm)
	c.Send(websocketEventAuthenticated())
	p.log(c, "201", perm.Pattern(), c.Id())
}

// The 'subscribe' event handler.
func (p *websocketProtocol) handleSubscribe(c *WebsocketClient, msg *Message) {
	// Getting data from payload...
	chanName, ok := msg.Get("channel").(string)
	if !ok || chanName == "" {
		// Bad request
		p.error(c, "400", errorBadRequest, "Invalid channel name")
		return
	}
	/*
		TODO: presence channels
		// Checking if current session have access to read from
		// this channel.
		if !c.IsAuthenticated() || !c.IsAllowed(chanName) {
			// Forbidden
			goto forbidden
			p.error(c, "403", errorForbidden, chanName)
			return
		}
	*/
	// Checking if channel exists...
	channel, err := c.vhost.Channel(chanName)
	if err != nil {
		// Channel not found
		p.error(c, "454", errorChannelNotFound, chanName)
		return
	}
	// Everything's fine, adding connection to subscribers and
	// sending an answer.
	channel.addSubscriber(c)
	c.Send(websocketEventSubscribed(chanName))
	p.log(c, "202", chanName, c.Id())
}

// The 'unsubscribe' event handler.
func (p *websocketProtocol) handleUnsubscribe(c *WebsocketClient, msg *Message) {
	// Getting data from payload...
	chanName, ok := msg.Get("channel").(string)
	if !ok || chanName == "" {
		// Bad request
		p.error(c, "400", errorBadRequest, "Invalid channel name")
		return
	}
	// Checking if channel exists and if current user is subscribing
	// this channel...
	channel, err := c.vhost.Channel(chanName)
	if err != nil || channel == nil {
		// Channel not found
		p.error(c, "454", errorChannelNotFound, chanName)
		return
	}
	if !channel.hasSubscriber(c) {
		// Not subscribed
		p.error(c, "453", errorNotSubscribed, chanName)
		return
	}
	// Unsubscribing from this channel
	channel.deleteSubscriber(c)
	c.Send(websocketEventUnsubscribed(chanName))
	p.log(c, "203", chanName, c.Id())
	return
}

// The 'broadcast' event handler.
func (p *websocketProtocol) handleBroadcast(c *WebsocketClient, msg *Message) {
	// Getting data from payload...
	chanName, _ := msg.Get("channel").(string)
	eventName, _ := msg.Get("event").(string)
	trigger, _ := msg.Get("trigger").(string)

	// Checking if message's payload is valid 
	if chanName == "" || eventName == "" {
		// Bad request
		p.error(c, "400", errorBadRequest, "Invalid payload params")
		return
	}
	// Data field is optional, so when it's empty we have to
	// assign an empty map.
	data, ok := msg.Get("data").(map[string]interface{})
	if !ok {
		data = map[string]interface{}{}
	}
	// Checking if user is allowed to publish on this channel
	//if !c.IsAuthenticated() || !c.IsAllowed(chanName) {
	//	// Forbidden
	//	p.error(c, "403", errorForbidden, chanName)
	//	return
	//}
	// Checking if channel exists...
	channel, err := c.vhost.Channel(chanName)
	if err != nil || channel == nil {
		// Channel not found
		p.error(c, "454", errorChannelNotFound, chanName)
		return
	}
	// ... and if client is subscribing it
	if !channel.hasSubscriber(c) {
		p.error(c, "453", errorNotSubscribed, chanName)
		return
	}
	// Extending data with sender and channel information before
	// passing it forward...
	data["sid"] = c.Id()
	data["channel"] = chanName
	// ... and broadcasting it to all subscribers.
	channel.Broadcast(&map[string]interface{}{eventName: data})
	p.log(c, "204", eventName, chanName, c.Id())
	// If the `trigger` param specified, then we have to send
	// an event to the backend agent.
	if trigger != "" {
		// TODO: Trigger backend job
	}
}

// The 'trigger' event handler.
func (p *websocketProtocol) handleTrigger(c *WebsocketClient, msg *Message) {
	// Getting data from payload...
	eventName, _ := msg.Get("event").(string)
	data, ok := msg.Get("data").(map[string]interface{})

	// Checking if message's payload is valid
	if eventName == "" || !ok {
		// Bad request
		return
	}
	// Checking if client is authenticated...
	if !c.IsAuthenticated() {
		// Forbidden
		return
	}
	// Extending data with sender information before passing
	// it forward...
	data["sid"] = c.Id()
	// ... and sending it to one of the agents.
	// TODO: send
}

// The 'pong' event handler.
func (p *websocketProtocol) handlePong(c *WebsocketClient, msg *Message) {
}

// The 'close' event handler.
func (p *websocketProtocol) handleClose(c *WebsocketClient, msg *Message) {
	// Just sending the confirmation
	c.Send(websocketEventClosed(c.Id()))
	p.log(c, "207", c.Id())
}

// Handles situation when requested event is not supported by
// the WebRocket Frontend Protocol.
func (p *websocketProtocol) notFound(c *WebsocketClient, msg *Message) {
	// Reply with the 'Bad request' error...
	p.error(c, "400", errorBadRequest, "Event not implemented")
}

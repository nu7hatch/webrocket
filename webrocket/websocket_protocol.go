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

// Helper for logging websocket handler's statuses.
func websocketStatusLog(c *WebsocketClient, status string, code int, msg string) {
	c.log.Printf("websocket[%s]: %d %s; %s", c.vhost.Path(), code, status, msg)
}

// Helper for logging protocol errors and and seding
// it to the client.
func websocketError(c *WebsocketClient, error string, code int, msg string) {
	payload := map[string]interface{}{
		"__error": map[string]interface{}{
			"code":   code,
			"status": error,
		},
	}
	c.Send(payload)
	websocketStatusLog(c, error, code, msg)
}

// Shorthand for returning handling a 'Bad Request' error.
func websocketBadRequestError(c *WebsocketClient, msg string) {
	websocketError(c, "Bad request", 400, msg)
}

// Available handlers for the websocket protocol.
var websocketProtocol = map[string]func(*WebsocketClient, *Message)(string,int){
	"broadcast":   websocketHandleBroadcast,
	"trigger":     websocketHandleTrigger,
	"subscribe":   websocketHandleSubscribe,
	"unsubscribe": websocketHandleUnsubscribe,
	"statChannel": websocketHandleStatChannel,
	"auth":        websocketHandleAuth,
	"close":       websocketHandleClose,
}

// websocketDispatch takes an incoming message and handles it
// in appropriate way according to the protocol specification.
func websocketDispatch(c *WebsocketClient, msg *Message) (string, int) {
	handlerFunc, ok := websocketProtocol[msg.Event()]
	if !ok {
		return "Bad request", 400
	}
	return handlerFunc(c, msg)
}

// The 'auth' event handler.
func websocketHandleAuth(c *WebsocketClient, msg *Message) (string, int) {
	// Getting data from payload...
	token, ok := msg.Get("token").(string)
	if !ok || token == "" {
		return "Bad request", 400
	}
	// Closing current session if authenticated...
	if c.IsAuthenticated() {
		c.clearSubscriptions()
		c.authenticate(nil) // ... and terminate this session
	}
	// Checking if given single access token exists
	perm, ok := c.vhost.ValidateSingleAccessToken(token)
	if !ok || perm == nil {
		return "Unauthorized", 402
	}
	// And if everything's fine then authenticate the client's
	// session and send a confirmation message. 
	c.authenticate(perm)
	c.Send(websocketEventAuthenticated())
	return "Authenticated", 201
}

// The 'subscribe' event handler.
func websocketHandleSubscribe(c *WebsocketClient, msg *Message) (string, int) {
	// Getting data from payload...
	chanName, ok := msg.Get("channel").(string)
	if !ok || chanName == "" {
		return "Bad request", 400
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
	channel, ok := c.vhost.Channel(chanName)
	if !ok {
		return "Channel not found", 454
	}
	// Everything's fine, adding connection to subscribers and
	// sending an answer.
	channel.Subscribe(c, true)
	c.Send(websocketEventSubscribed(chanName))
	return "Subscribed", 202
}

// The 'unsubscribe' event handler.
func websocketHandleUnsubscribe(c *WebsocketClient, msg *Message) (string, int) {
	// Getting data from payload...
	chanName, ok := msg.Get("channel").(string)
	if !ok || chanName == "" {
		return "Bad request", 400
	}
	// Checking if channel exists and if current user is subscribing
	// this channel...
	channel, ok := c.vhost.Channel(chanName)
	if !ok {
		return "Channel not found", 454
	}
	if !channel.hasSubscriber(c) {
		return "Not subscribed", 453
	}
	// Unsubscribing from this channel
	channel.Subscribe(c, false)
	c.Send(websocketEventUnsubscribed(chanName))
	return "Unsubscribed", 203
}

// The 'statChannel' event handler.
func websocketHandleStatChannel(c *WebsocketClient, msg *Message) (string, int) {
	// Getting data from payload...
	chanName, ok := msg.Get("channel").(string)
	if !ok || chanName == "" {
		return "Bad request", 400
	}
	// Checking if channel exists
	channel, ok := c.vhost.Channel(chanName)
	if !ok {
		return "Channel not found", 454
	}
	// TODO: check if authenticated
	// Sending channel information
	c.Send(websocketEventStatChannel(channel))
	return "Unsubscribed", 203
}

// The 'broadcast' event handler.
func websocketHandleBroadcast(c *WebsocketClient, msg *Message) (string, int) {
	// Getting data from payload...
	chanName, _ := msg.Get("channel").(string)
	eventName, _ := msg.Get("event").(string)
	trigger, _ := msg.Get("trigger").(string)
	// Checking if message's payload is valid 
	if chanName == "" || eventName == "" {
		return "Bad request", 400
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
	// If trigger specified, the client have to be authenticated
	if trigger != "" && !c.IsAuthenticated() {
		return "Forbidden", 403
	}
	// Checking if channel exists...
	channel, ok := c.vhost.Channel(chanName)
	if !ok {
		return "Channel not found", 454
	}
	// ... and if client is subscribing it
	if !channel.hasSubscriber(c) {
		return "Not subscribed", 453
	}
	// Extending data with sender and channel information before
	// passing it forward...
	data["sid"] = c.Id()
	data["channel"] = chanName
	// ... and broadcasting it to all subscribers.
	channel.Broadcast(&map[string]interface{}{eventName: data})
	// If the `trigger` param specified, then we have to send
	// an event to the backend agent.
	if trigger != "" {
		// Making sure that everything's fine
		if c.vhost == nil || c.vhost.ctx == nil || c.vhost.ctx.backend == nil {
			return "Internal error", 597
		}
		backend := c.vhost.ctx.backend
		// Triggering...
		backend.Trigger(c.vhost, &map[string]interface{}{trigger: data})
	}
	return "Broadcasted", 204
}

// The 'trigger' event handler.
func websocketHandleTrigger(c *WebsocketClient, msg *Message) (string, int) {
	// Getting data from payload...
	eventName, _ := msg.Get("event").(string)
	data, ok := msg.Get("data").(map[string]interface{})
	// Checking if message's payload is valid
	if eventName == "" || !ok {
		return "Bad request", 400
	}
	// Checking if client is authenticated...
	if !c.IsAuthenticated() {
		return "Forbidden", 403
	}
	// Making sure that everything's fine
	if c.vhost == nil || c.vhost.ctx == nil || c.vhost.ctx.backend == nil {
		return "Internal error", 597
	}
	backend := c.vhost.ctx.backend	
	// Extending data with sender information before passing
	// it forward...
	data["sid"] = c.Id()
	// ... and sending it to one of the agents.
	backend.Trigger(c.vhost, &map[string]interface{}{eventName: data})
	return "Triggered", 205
}

// The 'close' event handler.
func websocketHandleClose(c *WebsocketClient, msg *Message) (string, int) {
	// Just sending the confirmation
	c.Send(websocketEventClosed(c.Id()))
	c.Kill()
	return "Disconnected", 207
}

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

// The '__statChannel' event's payload.
func websocketEventStatChannel(channel *Channel) map[string]interface{} {
	return map[string]interface{}{
		"__statChannel": map[string]interface{}{
			"channel":          channel.name,
			"subscribersCount": len(channel.subscribers),
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

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
	"io"
	"net/http"
	"sync"
	"websocket"
)

// websocketHandler is a wrapper for the standard `websocket.Handler`
// providing some thread safety tricks and access to related vhost.
type websocketHandler struct {
	// Wrapped websocket handler.
	handler websocket.Handler
	// Endpoint to which the handler belogns.
	endpoint *WebsocketEndpoint
	// List of active connections.
	conns map[string]*WebsocketConnection
	// Whether the handler is alive or not.
	alive bool
	// Related vhost.
	vhost *Vhost
	// Internal semaphore.
	mtx sync.Mutex
}

// Internal constructor
// -----------------------------------------------------------------------------

// newWebsocketHandler creates a new handler for websocket connections.
//
// vhost    - The related vhost.
// endpoint - The parent websocket endpoint.
//
// Returns new handler.
func newWebsocketHandler(vhost *Vhost, endpoint *WebsocketEndpoint) (h *websocketHandler) {
	h = &websocketHandler{
		vhost:    vhost,
		alive:    true,
		endpoint: endpoint,
		conns:    make(map[string]*WebsocketConnection),
	}
	h.handler = websocket.Handler(func(ws *websocket.Conn) {
		h.handle(ws)
	})
	return h
}

// Internal
// -----------------------------------------------------------------------------

// addConn appends given connection to the active connections stack.
// Threadsafe, called from the internal handle function which is spawned
// into goroutine per each connected client.
//
// c - The websocket connection to be added to the stack.
//
func (h *websocketHandler) addConn(c *WebsocketConnection) {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	h.conns[c.Id()] = c
}

// deleteConn rrmoves given connection from the active connections stack.
// Threadsafe, called from the internal handle function which is spawned
// into goroutine per each connected client.
//
// c - The websocket connection to be removed from the stack.
//
func (h *websocketHandler) deleteConn(c *WebsocketConnection) {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	delete(h.conns, c.Id())
}

// disconnectAll closes all active connections. Not threadsafe, called only
// from the internal Kill function.
func (h *websocketHandler) disconnectAll() {
	for _, c := range h.conns {
		c.Kill()
	}
}

// handle implements an event loop for handling single websocket connection.
// Each incoming connection has it running in its own goroutine.
//
// ws - The raw websocket connection to be handled.
//
func (h *websocketHandler) handle(ws *websocket.Conn) {
	c := newWebsocketConnection(ws)
	h.addConn(c)
	defer h.deleteConn(c)
	h.logStatus(c, "Connected", 200, "")
	for {
		if !h.IsAlive() {
			break
		}
		if msg, err := c.Receive(); err == nil && msg != nil {
			h.dispatch(c, msg)
		} else if err == io.EOF {
			// End of file reached, terminating this connection...
			c.Kill()
			break
		} else {
			h.logStatus(c, "Bad request", 400, "")
		}
	}
}

// dispatch takes given message and handles it in appropriate way according
// to the Websocket Frontend Protocol specification.
//
// c   - The websocket connection to dispatch against.
// msg - A message to be dispatched.
//
func (h *websocketHandler) dispatch(c *WebsocketConnection, msg *WebsocketMessage) {
	var status string
	var code int
	// Route to the appropriate handler.
	switch msg.Event() {
	case "broadcast":
		status, code = h.handleBroadcast(c, msg)
	case "trigger":
		status, code = h.handleTrigger(c, msg)
	case "subscribe":
		status, code = h.handleSubscribe(c, msg)
	case "unsubscribe":
		status, code = h.handleUnsubscribe(c, msg)
	case "auth":
		status, code = h.handleAuth(c, msg)
	case "close":
		status, code = h.handleClose(c, msg)
	default:
		status, code = "Bad request", 400
	}
	// Log status code.
	h.logStatus(c, status, code, msg.JSON())
}

// logStatus writes specified status information to the logs. The 3xx
// statuses shall be logged only when debug mode is enabled. If clients
// specifies that wants to get error messages, then it should send such
// one in case of handling a 4xx or 5xx status.
//
// c      - Related websocket connection.
// status - A textual status message.
// code   - Status code.
// msg    - Related message.
//
// Example:
//
//     h.logStatus(c, "Bad request", 400, handledMessage)
//
func (h *websocketHandler) logStatus(c *WebsocketConnection, status string,
	code int, msg string) {
	switch {
	case code >= 400:
		// TODO: make the answers only when the client's debug mode is
		// enabled or some 'optional errors' flag enabled or smth...
		c.Send(map[string]interface{}{
			"__error": map[string]interface{}{
				"code":   code,
				"status": status,
			},
		})
	case code >= 300 && code < 400:
		// Log information statuses only when debug mode is enabled.
		// TODO: log after adding a debug mode...
		return
	case code < 300:
		// Nothing to do, just go to logging...
	}
	if h.vhost == nil {
		// Should never happen, but better safe than sorry!
		return
	}
	h.endpoint.log.Printf("websocket[%s]: %d %s; %s", h.vhost.Path(), code, status, msg)
}

// Websocket Frontent Protocol handlers
// -----------------------------------------------------------------------------

// handleAuth is a handler for the 'authenticate' Websocket Frontend
// Protocol event.
func (h *websocketHandler) handleAuth(c *WebsocketConnection,
	msg *WebsocketMessage) (string, int) {
	// {
	//     "token": "access token..."
	// }
	var ok bool
	var token string
	var perm *Permission

	if token, ok = msg.Get("token").(string); !ok || token == "" {
		// No token specified, invalid payload!
		return "Bad request", 400
	}
	if c.IsAuthenticated() {
		// Close current session if authenticated.
		c.reauthenticate(nil)
	}
	if perm, ok = h.vhost.ValidateSingleAccessToken(token); !ok || perm == nil {
		// No such sigle access token, access denied!
		return "Unauthorized", 402
	}
	c.authenticate(perm)
	return "Authenticated", 201
}

// handleSubscribe is a handler for the 'subscribe' Websocket Frontend
// Protocol event.
//
// c   - Related websocket connection.
// msg - The message to be handled.
//
// Returns status message and code.
func (h *websocketHandler) handleSubscribe(c *WebsocketConnection,
	msg *WebsocketMessage) (string, int) {
	// {
	//     "channel": "channel name...",
	//     "hidden":  true, // or false
	//     "data": {...}
	// }
	var err error
	var chanName string
	var hidden, ok bool
	var data map[string]interface{}
	var channel *Channel

	if chanName, ok = msg.Get("channel").(string); chanName == "" {
		// Channel name not found, invalid payload!
		return "Bad request", 400
	}
	if hidden, ok = msg.Get("hidden").(bool); !ok {
		// No hidden option specified, setting to false by default.
		hidden = false
	}
	if data, ok = msg.Get("data").(map[string]interface{}); !ok {
		// No user data specified, making empty one by default.
		data = make(map[string]interface{})
	}
	if channel, err = h.vhost.Channel(chanName); err != nil {
		// Nope, channel not found!
		return "Channel not found", 454
	}
	if channel.IsPrivate() && !c.IsAllowed(chanName) {
		// Can't operate on this channel, access denied!
		return "Forbidden", 403
	}
	channel.subscribe(c, hidden, data)
	return "Subscribed", 202
}

// handleUnsubscribe is a handler for the 'unsubscribe' Websocket Frontend
// Protocol event.
//
// c   - Related websocket connection.
// msg - The message to be handled.
//
// Returns status message and code.
func (h *websocketHandler) handleUnsubscribe(c *WebsocketConnection,
	msg *WebsocketMessage) (string, int) {
	// {
	//     "channel": "channel name...",
	//     "data": {...}
	// }
	var ok bool
	var err error
	var chanName string
	var data map[string]interface{}
	var channel *Channel

	if chanName, ok = msg.Get("channel").(string); chanName == "" {
		// Channel name not found, invalid payload!
		return "Bad request", 400
	}
	if data, ok = msg.Get("data").(map[string]interface{}); !ok {
		// No user data specified, making empty one by default.
		data = make(map[string]interface{})
	}
	if channel, err = h.vhost.Channel(chanName); err != nil {
		// Nope, channel not found!
		return "Channel not found", 454
	}
	if !channel.HasSubscriber(c) {
		// This guy is not subscribing this channel!
		return "Not subscribed", 453
	}
	channel.unsubscribe(c, data, true)
	return "Unsubscribed", 203
}

// handleBroadcast is a handler for the 'broadcast' Websocket Frontend
// Protocol event.
//
// c   - Related websocket connection.
// msg - The message to be handled.
//
// Returns status message and code.
func (h *websocketHandler) handleBroadcast(c *WebsocketConnection,
	msg *WebsocketMessage) (string, int) {
	// {
	//     "channel": "channel name...",
	//     "event": "event name...",
	//     "trigger": "backend event name...",
	//     "data": {...}
	// }
	var ok bool
	var err error
	var chanName, eventName, triggerName string
	var data map[string]interface{}
	var channel *Channel

	if chanName, ok = msg.Get("channel").(string); chanName == "" {
		// Channel name not found, invalid payload!
		return "Bad request", 400
	}
	if eventName, ok = msg.Get("event").(string); eventName == "" {
		// Event name not found, invalid payload!
		return "Bad request", 400
	}
	if data, ok = msg.Get("data").(map[string]interface{}); !ok {
		// No user data specified, making empty one by default.
		data = make(map[string]interface{})
	}
	if triggerName, ok = msg.Get("trigger").(string); !ok {
		// No trigger by default.
		triggerName = ""
	}
	if channel, err = h.vhost.Channel(chanName); err != nil {
		// Nope, channel not found!
		return "Channel not found", 454
	}
	if !channel.HasSubscriber(c) {
		// Can't broadcast on the channel without subscribing it!
		return "Not subscribed", 453
	}
	if triggerName != "" && !c.IsAuthenticated() { // FIXME: Backend should have permissions too!
		// Can't trigger, access denied!
		return "Forbidden", 403
	}
	// Extending data with sender and channel information before
	// passing it forward...
	data["sid"] = c.Id()
	data["channel"] = chanName
	channel.Broadcast(map[string]interface{}{eventName: data})
	// If the `trigger` param specified, then we have to send an event
	// to the backend agent.
	if triggerName != "" {
		if h.vhost == nil || h.vhost.ctx == nil || h.vhost.ctx.backend == nil {
			// Should never happen, but you know... never say never :)
			return "Internal error", 597
		}
		backend := h.vhost.ctx.backend
		backend.Trigger(h.vhost, &map[string]interface{}{triggerName: data})
	}
	return "Broadcasted", 204
}

// handleTrigger is a handler for the 'trigger' Websocket Frontend Protocol
// event.
//
// c   - Related websocket connection.
// msg - The message to be handled.
//
// Returns status message and code.
func (h *websocketHandler) handleTrigger(c *WebsocketConnection,
	msg *WebsocketMessage) (string, int) {
	// {
	//     "event": "event name...",
	//     "data": {...}
	// }
	var ok bool
	var eventName string
	var data map[string]interface{}

	if eventName, ok = msg.Get("event").(string); eventName == "" {
		// Event name not found, invalid payload!
		return "Bad request", 400
	}
	if data, ok = msg.Get("data").(map[string]interface{}); !ok {
		// No user data specified, making empty one by default.
		data = make(map[string]interface{})
	}
	if !c.IsAuthenticated() { // FIXME: Backend should have permissions too!
		// Can't trigger, access denied!
		return "Forbidden", 403
	}
	if h.vhost == nil || h.vhost.ctx == nil || h.vhost.ctx.backend == nil {
		// Should never happen... i hope...
		return "Internal error", 597
	}
	// Extending data with sender information before passing
	// it forward...
	data["sid"] = c.Id()
	backend := h.vhost.ctx.backend
	backend.Trigger(h.vhost, &map[string]interface{}{eventName: data})
	return "Triggered", 205
}

// handleClose is a handler for the 'close' Websocket Frontend Protocol event.
//
// c   - Related websocket connection.
// msg - The message to be handled.
//
// Returns status message and code.
func (h *websocketHandler) handleClose(c *WebsocketConnection,
	msg *WebsocketMessage) (string, int) {
	c.Kill()
	return "Disconnected", 207
}

// Exported
// -----------------------------------------------------------------------------

// IsAlive returns whether the handler is alive or not. Threadsafe, Can be
// called from many connections' goroutines and depends on the Kill function
// calls.
func (h *websocketHandler) IsAlive() bool {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	return h.alive
}

// Kill stops execution of this handler and disconnects all connected clients.
// Threadsafe, can be called only from the websocket endpoint, but the IsAlive
// function's result depends on it.
func (h *websocketHandler) Kill() {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	if h.alive {
		h.alive = false
		h.disconnectAll()
	}
}

// ServeHTTP extends standard websocket.Handler implementation of http.Handler
// interface.
//
// w - The HTTP response writer.
// r - The request to be handled.
//
func (h *websocketHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if h.IsAlive() {
		h.handler.ServeHTTP(w, req)
	}
}

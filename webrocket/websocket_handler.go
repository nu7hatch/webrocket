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

import (
	"io"
	"net/http"
	"sync"
	"encoding/json"
	"websocket"
)

// websocketHandler is a wrapper for the standard `websocket.Handler`
// providing some thread safety tricks and access to related vhost.
type websocketHandler struct {
	handler   websocket.Handler
	isRunning bool
	vhost     *Vhost
	mtx       sync.Mutex
}

// Creates new handler.
func newWebsocketHandler(vhost *Vhost) *websocketHandler {
	h := &websocketHandler{vhost: vhost, isRunning: false}
	h.handler = websocket.Handler(func(ws *websocket.Conn) { h.handle(ws) })
	return h
}

// handle is an event loop for handling single websocket connection. 
func (h *websocketHandler) handle(ws *websocket.Conn) {
	c := newWebsocketClient(h.vhost, ws)
	defer c.kill()
	// New connection established, so we have to send the '__connected'
	// event to the client.
	c.Send(websocketEventConnected(c.Id()))
	websocketStatusLog(c, "Connected", 200, "")
	for {
		// Break if client is dead
		if !c.IsAlive() {
			break
		}
		// Break if handler has been stopped
		if !h.IsRunning() {
			c.kill()
			break
		}
		// Handle connection
		h.doHandle(c)
	}
}

// doHandle is actual handler used in by an event loop defined
// in the `handle` func.
//
// TODO: make this handler non-blocking handler
//
func (h *websocketHandler) doHandle(c *WebsocketClient) {
	var recv map[string]interface{}
	err := websocket.JSON.Receive(c.Conn, &recv)
	if err != nil {
		if err == io.EOF {
			// End of file reached, which means that connection
			// has been closed.
			websocketError(c, "Connection closed", 598, "")
			c.kill()
			return
		}
		// Any other case means that data we have received
		// is invalid.
		websocketBadRequestError(c, "")
		return
	}
	msg, err := newMessage(recv)
	msgstr, _ := json.Marshal(recv)
	if err != nil {
		// Message couldn't be parsed due to invalid JSON format.
		websocketBadRequestError(c, string(msgstr))
		return
	}
	// Finally, if everything's cool, dispatch the message
	// and check it's response status.
	status, code, keepgoing := websocketDispatch(c, msg)
	if code >= 400 {
		websocketError(c, status, code, string(msgstr))
	} else {
		websocketStatusLog(c, status, code, string(msgstr))
	}
	// If returned value is false, that means the connection
	// has been closed and loop should be terminated.
	if !keepgoing {
		c.kill()
	}
}

// Stops execution of this handler.
func (h *websocketHandler) stop() {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	h.isRunning = false
}

// Enables execution status  of this handler. 
func (h *websocketHandler) start() {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	h.isRunning = true
}

// Returns true if this handler is still alive.
func (h *websocketHandler) IsRunning() bool {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	return h.isRunning
}

// ServeHTTP extends standard websocket.Handler implementation
// of http.Handler interface.
func (h *websocketHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if h.IsRunning() {
		h.handler.ServeHTTP(w, req)
	}
}

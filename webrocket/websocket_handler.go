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
	"net/http"
	"sync"
	"websocket"
)

// websocketHandler is a wrapper for the standard `websocket.Handler`
// providing some thread safety tricks and access to related vhost.
type websocketHandler struct {
	handler  websocket.Handler
	conns    map[string]*WebsocketClient
	alive    bool
	vhost    *Vhost
	mtx      sync.Mutex
}

// Creates new handler.
func newWebsocketHandler(vhost *Vhost) (h *websocketHandler) {
	h = &websocketHandler{
		vhost:   vhost,
		alive:   true,
		conns:   make(map[string]*WebsocketClient),
	}
	h.handler = websocket.Handler(func(ws *websocket.Conn) {
		h.handle(ws)
	})
	return h
}

// Puts given connection to the stack.
func (h *websocketHandler) addConn(c *WebsocketClient) {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	h.conns[c.Id()] = c
}

// Removes given connection from the stack.
func (h *websocketHandler) deleteConn(c *WebsocketClient) {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	delete(h.conns, c.Id())
}

// Closes all connection from the handler's stack.
func (h *websocketHandler) disconnectAll() {
	for _, c := range h.conns {
		c.Kill()
	}
}

// handle is an event loop for handling single websocket connection. 
func (h *websocketHandler) handle(ws *websocket.Conn) {
	c := newWebsocketClient(h.vhost, ws)
	h.addConn(c)
	defer h.deleteConn(c)
	// New connection established, so we have to send the '__connected'
	// event to the client.
	c.Send(websocketEventConnected(c.Id()))
	websocketStatusLog(c, "Connected", 200, "")
	for {
		if !h.IsAlive() {
			// Break if handler has been stopped.
			break
		}
		msg := c.Receive()
		if msg == nil {
			// Connection closed.
			break
		}
		// Dispatch the message...
		status, code := websocketDispatch(c, msg)
		// ... and log status info.
		if code >= 400 {
			websocketError(c, status, code, msg.JSON())
		} else {
			websocketStatusLog(c, status, code, msg.JSON())
		}
	}
}

// Returns true if this handler is still alive.
func (h *websocketHandler) IsAlive() bool {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	return h.alive
}

// Stops execution of this handler.
func (h *websocketHandler) Kill() {
	h.mtx.Lock()
	defer h.mtx.Unlock()
	h.alive = false
	h.disconnectAll()
}

// ServeHTTP extends standard websocket.Handler implementation
// of http.Handler interface.
func (h *websocketHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if h.IsAlive() {
		h.handler.ServeHTTP(w, req)
	}
}

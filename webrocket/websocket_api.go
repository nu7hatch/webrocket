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
	"errors"
	"fmt"
)

// Handles API calls via frontend WebSockets protocol.
type websocketAPI struct{}

// Dispatch matches message given event with protocol and
// executes proper operation. Returns information if the
// conneciton should be still maintained, and eventual error.
func (api *websocketAPI) Dispatch(c *wsConn, msg *Message) (bool, error) {
	switch msg.Event {
	case "broadcast":
		return true, api.doBroadcast(c, msg.Data)
	case "subscribe":
		return true, api.doSubscribe(c, msg.Data)
	case "unsubscribe":
		return true, api.doUnsubscribe(c, msg.Data)
	case "auth":
		return true, api.doAuthenticate(c, msg.Data)
	case "close":
		return false, api.doClose(c)
	}
	return true, api.notFound(c, msg.Event)
}

// A helper for quick handling error responses.
func (api *websocketAPI) Error(c *wsConn, payload map[string]interface{}) error {
	err := errors.New(fmt.Sprintf("ERR_%s", payload["id"]))
	c.vhost.Log.Printf("ws[%s]: %s", c.vhost.path, err.Error())
	c.send(map[string]interface{}{"__error": payload})
	return err
}

// Authenticates session for the specified user.
func (api *websocketAPI) doAuthenticate(c *wsConn, data interface{}) error {
	payload, ok := data.(map[string]interface{})
	if !ok {
		// INVALID_PAYLOAD
		return api.Error(c, ErrInvalidPayload)
	}
	username, ok := payload["user"].(string)
	if !ok {
		// AUTH_INVALID_USERNAME
		return api.Error(c, ErrInvalidUserName)
	}
	secret, ok := payload["secret"].(string)
	if !ok {
		secret = ""
	}
	user, ok := c.vhost.GetUser(username)
	if !ok {
		// AUTH_USER_NOT_FOUND
		return api.Error(c, ErrUserNotFound)
	}
	ok = user.Authenticate(secret)
	if !ok {
		// AUTH_INVALID_CREDENTIALS
		c.session = nil
		return api.Error(c, ErrInvalidCredentials)
	}
	c.session = user
	err := c.send(map[string]interface{}{
		"__authenticated": map[string]interface{}{
			"user": user.Name,
			"permission": user.Permission,
		},
	})
	if err != nil {
		// NOT_SENT
		return err
	}
	c.vhost.Log.Printf("ws[%s]: AUTHENTICATED user='%s'", c.vhost.path, username)
	return nil
}

// Subscribes client to the specified channel.
func (api *websocketAPI) doSubscribe(c *wsConn, data interface{}) error {
	user := c.session
	if user == nil || !user.IsAllowed(PermRead) {
		// ACCESS_DENIED
		return api.Error(c, ErrAccessDenied)
	}
	payload, ok := data.(map[string]interface{})
	if !ok {
		// INVALID_PAYLOAD
		return api.Error(c, ErrInvalidPayload)
	}
	chanName, ok := payload["channel"].(string)
	if !ok || len(chanName) == 0 {
		// INVALID_CHANNEL_NAME
		return api.Error(c, ErrInvalidChannelName)
	}
	ch := c.vhost.GetOrCreateChannel(chanName)
	ch.subscribe <- subscription{c, true}
	err := c.send(map[string]interface{}{
		"__subscribed": map[string]interface{}{
			"channel": chanName,
		},
	})
	if err != nil {
		// NOT_SENT
		return err
	}
	c.vhost.Log.Printf("ws[%s]: SUBSCRIBED channel='%s'", c.vhost.path, chanName)
	return nil
}

// Unsubscribes client from the specified channnel.
func (api *websocketAPI) doUnsubscribe(c *wsConn, data interface{}) error {
	user := c.session
	if user == nil || !user.IsAllowed(PermRead) {
		// ACCESS_DENIED
		return api.Error(c, ErrAccessDenied)
	}
	payload, ok := data.(map[string]interface{})
	if !ok {
		// INVALID_PAYLOAD
		return api.Error(c, ErrInvalidPayload)
	}
	chanName, ok := payload["channel"].(string)
	if !ok || len(chanName) == 0 {
		// INVALID_CHANNEL_NAME
		return api.Error(c, ErrInvalidChannelName)
	}
	ch, ok := c.vhost.GetChannel(chanName)
	if !ok {
		// CHANNEL_NOT_FOUND
		return api.Error(c, ErrChannelNotFound)
	}
	ch.subscribe <- subscription{c, false}
	c.vhost.Log.Printf("ws[%s]: UNSUBSCRIBED channel='%s'", c.vhost.path, chanName)
	return nil
}

// Broadcasts and triggers client events with specified data on
// given channels.
func (api *websocketAPI) doBroadcast(c *wsConn, data interface{}) error {
	user := c.session
	if user == nil || !user.IsAllowed(PermWrite) {
		// ACCESS_DENIED
		return api.Error(c, ErrAccessDenied)
	}
	payload, ok := data.(map[string]interface{})
	if !ok {
		// INVALID_PAYLOAD
		return api.Error(c, ErrInvalidPayload)
	}
	event, ok := payload["event"].(string)
	if !ok || len(event) == 0 {
		// INVALID_EVENT_NAME
		return api.Error(c, ErrInvalidEventName)
	}
	chanName, ok := payload["channel"].(string)
	if !ok || len(chanName) == 0 {
		// INVALID_CHANNEL_NAME
		return api.Error(c, ErrInvalidChannelName)
	}
	ch, ok := c.vhost.GetChannel(chanName)
	if !ok {
		// CHANNEL_NOT_FOUND
		return api.Error(c, ErrChannelNotFound)
	}
	ch.broadcast <- data
	c.vhost.Log.Printf("ws[%s]: BROADCASTED event='%s' channel='%s'", c.vhost.path, event, chanName)
	return nil
}

// Safely closes connection.
func (api *websocketAPI) doClose(c *wsConn) error {
	c.session = nil
	c.unsubscribeAll()
	c.Close()
	c.vhost.Log.Printf("ws[%s]: CLOSED", c.vhost.path)
	return nil
}

// Handles error when operation specified in payload is not
// defined in the API.
func (api *websocketAPI) notFound(c *wsConn, event string) error {
	payload := ErrUndefinedEvent
	payload["event"] = event
	c.vhost.Log.Printf("ws[%s]: NOT_FOUND event='%s'", c.vhost.path, event)
	return api.Error(c, payload)
}

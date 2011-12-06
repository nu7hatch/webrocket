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

// Shortcut for defining payload data.
type Payload map[string]string

// Handles API calls via frontend WebSockets protocol.
type websocketAPI struct{}

// Predefined error payloads.
var (
	ErrInvalidPayload     = Payload{"err": "INVALID_PAYLOAD"}
	ErrAccessDenied       = Payload{"err": "ACCESS_DENIED"}
	ErrInvalidUserName    = Payload{"err": "INVALID_USER_NAME"}
	ErrUserNotFound       = Payload{"err": "USER_NOT_FOUND"}
	ErrInvalidCredentials = Payload{"err": "INVALID_CREDENTIALS"}
	ErrInvalidChannelName = Payload{"err": "INVALID_CHANNEL_NAME"}
	ErrChannelNotFound    = Payload{"err": "CHANNEL_NOT_FOUND"}
	ErrInvalidEventName   = Payload{"err": "INVALID_EVENT_NAME"}
	ErrUndefinedEvent     = Payload{"err": "UNDEFINED_EVENT"}
)

// Dispatch matches message given event with protocol and
// executes proper operation. Returns information if the
// conneciton should be still maintained, and eventual error.
func (api *websocketAPI) Dispatch(c *conn, msg *Message) (bool, error) {
	switch msg.Event {
	case "broadcast":
		return true, api.doBroadcast(c, msg.Data)
	case "authenticate":
		return true, api.doAuthenticate(c, msg.Data)
	case "subscribe":
		return true, api.doSubscribe(c, msg.Data)
	case "unsubscribe":
		return true, api.doUnsubscribe(c, msg.Data)
	case "logout":
		return true, api.doLogout(c)
	case "disconnect":
		return false, api.doDisconnect(c)
	}
	return true, api.notFound(c, msg.Event)
}

// A helper for quick handling error responses.
func (api *websocketAPI) error(c *conn, payload map[string]string) error {
	err := errors.New(fmt.Sprintf("ERR_%s", payload["err"]))
	c.vhost.Log.Printf("ws[%s]: %s", c.vhost.path, err.Error())
	c.send(payload)
	return err
}

// Authenticates session for the specified user.
//
// Example:
//
//     { "authenticate": {"user": "joe", "secret": "53cr37"}}
//
// Payload:
//
// * `user` - name of the configured user you want to authenticate (required)
// * `secret` - authentication secret for specified user (optional)
//
// Errors:
//
// * `INVALID_CREDENTIALS` - returned when given secret is invalid
// * `USER_NOT_FOUND` - returned when given user does not exist
// * `INVALID_USER_NAME` - returned when no username given or its format is invalid
// * `INVALID_PAYLOAD` - returned when data format is invalid
//
func (api *websocketAPI) doAuthenticate(c *conn, data interface{}) error {
	payload, ok := data.(map[string]interface{})
	if !ok {
		// INVALID_PAYLOAD
		return api.error(c, ErrInvalidPayload)
	}
	username, ok := payload["user"].(string)
	if !ok {
		// AUTH_INVALID_USERNAME
		return api.error(c, ErrInvalidUserName)
	}
	secret, ok := payload["secret"].(string)
	if !ok {
		secret = ""
	}
	user, ok := c.vhost.GetUser(username)
	if !ok {
		// AUTH_USER_NOT_FOUND
		return api.error(c, ErrUserNotFound)
	}
	ok = user.Authenticate(secret)
	if !ok {
		// AUTH_INVALID_CREDENTIALS
		return api.error(c, ErrInvalidCredentials)
	}
	c.session = user
	err := c.send(Payload{"authenticated": username})
	if err != nil {
		// NOT_SENT
		return err
	}
	c.vhost.Log.Printf("ws[%s]: AUTHENTICATED user='%s'", c.vhost.path, username)
	return nil
}

// Subscribes client to the specified channel.
//
// Example:
//
//     {"subscribe": {"channel": "hello"}}
//
// Payload:
//
// * `channel` - name of channel you want to subscribe, not existing
//               channels are created automatically (required)
//     
// Errors:
//
// * `INVALID_CHANNEL_NAME` - returned when no channel name given or when given name is invalid
// * `ACCESS_DENIED` - returned when current session is not authenticated for reading
// * `INVALID_PAYLOAD` - returned when payload format is invalid
//
func (api *websocketAPI) doSubscribe(c *conn, data interface{}) error {
	user := c.session
	if user == nil || !user.IsAllowed(PermRead) {
		// ACCESS_DENIED
		return api.error(c, ErrAccessDenied)
	}
	payload, ok := data.(map[string]interface{})
	if !ok {
		// INVALID_PAYLOAD
		return api.error(c, ErrInvalidPayload)
	}
	chanName, ok := payload["channel"].(string)
	if !ok || len(chanName) == 0 {
		// INVALID_CHANNEL_NAME
		return api.error(c, ErrInvalidChannelName)
	}
	ch := c.vhost.GetOrCreateChannel(chanName)
	ch.subscribe <- subscription{c, true}
	err := c.send(Payload{"subscribed": chanName})
	if err != nil {
		// NOT_SENT
		return err
	}
	c.vhost.Log.Printf("ws[%s]: SUBSCRIBED channel='%s'", c.vhost.path, chanName)
	return nil
}

// Unsubscribes client from the specified channnel.
//
// Example:
//
//     {"unsubscribe": {"channel": "channel-name"}}
//
// Payload:
//             
// * `channel` - name of channel you want to unsubscribe (required)
//     
// Errors:
//
// * `INVALID_CHANNEL_NAME` - returned when no channel name given or when given name is invalid
// * `CHANNEL_NOT_FOUND` - returned when given channel doesn't exist
// * `ACCESS_DENIED` - returned when current session is not authenticated for reading
// * `INVALID_PAYLOAD` - returned when payload format is invalid
//
func (api *websocketAPI) doUnsubscribe(c *conn, data interface{}) error {
	user := c.session
	if user == nil || !user.IsAllowed(PermRead) {
		// ACCESS_DENIED
		return api.error(c, ErrAccessDenied)
	}
	payload, ok := data.(map[string]interface{})
	if !ok {
		// INVALID_PAYLOAD
		return api.error(c, ErrInvalidPayload)
	}
	chanName, ok := payload["channel"].(string)
	if !ok || len(chanName) == 0 {
		// INVALID_CHANNEL_NAME
		return api.error(c, ErrInvalidChannelName)
	}
	ch, ok := c.vhost.GetChannel(chanName)
	if !ok {
		// CHANNEL_NOT_FOUND
		return api.error(c, ErrChannelNotFound)
	}
	ch.subscribe <- subscription{c, false}
	err := c.send(Payload{"unsubscribed": chanName})
	if err != nil {
		// NOT_SENT
		return err
	}
	c.vhost.Log.Printf("ws[%s]: UNSUBSCRIBED channel='%s'", c.vhost.path, chanName)
	return nil
}

// Broadcasts and triggers client events with specified data on given channels.
//
// Example:
//
//     {"broadcast": {"event": "hello", "channel": "world", data: {"x": 1}}}
//
// Payload:
//
// * `event` - name of the event which will be triggerred on the client side (required)
// * `channel` - channel have to exist (required)
// * `data` - data to publish (optional)
//
// Errors:
//
// * `INVALID_EVENT_NAME` - returned when no event name given when given name is invalid
// * `INVALID_CHANNEL_NAME` - returned when no channel name given
// * `CHANNEL_NOT_FOUND` - returned when given channel doesn't exist
// * `ACCESS_DENIED` - returned when current session is not authenticated for writing
// * `INVALID_PAYLOAD` - returned when payload format is invalid
//
func (api *websocketAPI) doBroadcast(c *conn, data interface{}) error {
	user := c.session
	if user == nil || !user.IsAllowed(PermWrite) {
		// ACCESS_DENIED
		return api.error(c, ErrAccessDenied)
	}
	payload, ok := data.(map[string]interface{})
	if !ok {
		// INVALID_PAYLOAD
		return api.error(c, ErrInvalidPayload)
	}
	event, ok := payload["event"].(string)
	if !ok || len(event) == 0 {
		// INVALID_EVENT_NAME
		return api.error(c, ErrInvalidEventName)
	}
	chanName, ok := payload["channel"].(string)
	if !ok || len(chanName) == 0 {
		// INVALID_CHANNEL_NAME
		return api.error(c, ErrInvalidChannelName)
	}
	ch, ok := c.vhost.GetChannel(chanName)
	if !ok {
		// CHANNEL_NOT_FOUND
		return api.error(c, ErrChannelNotFound)
	}
	ch.broadcast <- data
	// broadcasting is fault tolerant, so we can skip err checking
	// on sending the answer.
	c.send(Payload{"broadcasted": chanName})
	c.vhost.Log.Printf("ws[%s]: BROADCASTED event='%s' channel='%s'", c.vhost.path, event, chanName)
	return nil
}

// Finishes current session and unsubscribes from all channels.
//
// Example:
//
//     {"logout": true}
//
// Errors:
//
// * `ACCESS_DENIED` - returned when current session is not authenticated
// * `INVALID_PAYLOAD` - returned when payload format is invalid
//
func (api *websocketAPI) doLogout(c *conn) error {
	user := c.session
	if user == nil || !user.IsAllowed(PermRead) {
		// ACCESS_DENIED
		return api.error(c, ErrAccessDenied)
	}
	c.unsubscribeAll()
	c.session = nil
	err := c.send(map[string]bool{"loggedOut": true})
	if err != nil {
		return err
	}
	c.vhost.Log.Printf("ws[%s]: LOGGED_OUT user='%s'", c.vhost.path, user.Name)
	return nil
}

// Safely closes connection.
//
// Example:
//
//     {"disconnect": true}
//
// Errors:
//
// * `INVALID_PAYLOAD` - returned when payload format is invalid
//
func (api *websocketAPI) doDisconnect(c *conn) error {
	c.unsubscribeAll()
	c.Close()
	return nil
}

// Handles error when operation specified in payload is not
// defined in the API.
func (api *websocketAPI) notFound(c *conn, event string) error {
	payload := Payload(ErrUndefinedEvent)
	payload["event"] = event
	return api.error(c, payload)
}

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
	"encoding/json"
	"errors"
)

// WebsocketMessage implements representation of the decoded message sent
// by the websocket client.
type WebsocketMessage struct {
	event string
	data  map[string]interface{}
}

// Internal constructors
// -----------------------------------------------------------------------------

// newWebsocketMessageFromJSON decodes JSON stream and generates maps it into
// WebsocketMessage instance. 
func newWebsocketMessageFromJSON(payload []byte) (msg *WebsocketMessage, err error) {
	var decoded map[string]interface{}
	if err = json.Unmarshal(payload, &decoded); err != nil {
		return
	}
	msg, err = newWebsocketMessage(decoded)
	return
}

// newWebsocketMessage converts given map into message instance. Valid message
// has format: {"event": {...}}. where first (the only) key is considered to
// be an event name, and it's value is a data's payload.
//
// If given map couldn't be converted into message then function will return
// an appropriate error.
//
// payload - A map to convert into a message.
//
// Examples
//
//     x = map[string]interface{}{
//        "foo": map[string]interface{}{
//            "bar": 1,
//        },
//     }
//     msg, _ = newWebsocketMessage(x)
//     println(msg.Event())
//     // => "foo"
//     println(msg.Get("bar"))
//     // => 1
//
// Returns new websocket message or error if something went wrong.
func newWebsocketMessage(payload map[string]interface{}) (msg *WebsocketMessage, err error) {
	if len(payload) != 1 {
		err = errors.New("invalid message format")
		return
	}
	msg = &WebsocketMessage{}
	for k := range payload {
		msg.event = k
	}
	var ok bool
	msg.data, ok = payload[msg.event].(map[string]interface{})
	if !ok {
		err = errors.New("invalid message data type")
	}
	return
}

// Exported
// -----------------------------------------------------------------------------

// Event returns an event name.
func (m *WebsocketMessage) Event() string {
	return m.event
}

// Data returns a data's payload.
func (m *WebsocketMessage) Data() map[string]interface{} {
	return m.data
}

// Get returns a value from the specified data field. If the key is not
// defined then nil will be returned.
//
// key - A key to find in the data
//
// Examples
//
//    foo, ok := msg.Get("foo").(string)
//    bar, ok := msg.Get("bar").(int)
//
// Returns value of the specified key.
func (m *WebsocketMessage) Get(key string) (value interface{}) {
	value, _ = m.data[key]
	return
}

// JSON returns message encoded into a JSON string.
func (m *WebsocketMessage) JSON() string {
	buf, _ := json.Marshal(&map[string]interface{}{m.event: m.data})
	return string(buf)
}

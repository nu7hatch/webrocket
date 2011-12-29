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
	"encoding/json"
	"errors"
)

// Message is a simple structure which keeps the incoming events
// information and data.
type Message struct {
	event string
	data  map[string]interface{}
}

// newMessageFromJSON decoded JSON stream and generates message
// from it. 
func newMessageFromJSON(payload []byte) (msg *Message, err error) {
	var decoded map[string]interface{}
	err = json.Unmarshal(payload, &decoded)
	if err != nil {
		return
	}
	msg, err = newMessage(decoded)
	return msg, err
}

// newMessage converts received map into message structure.
func newMessage(payload map[string]interface{}) (msg *Message, err error) {
	if len(payload) != 1 {
		err = errors.New("Invalid message format")
		return
	}
	msg = &Message{}
	for k := range payload {
		msg.event = k
	}
	var ok bool
	msg.data, ok = payload[msg.event].(map[string]interface{})
	if !ok {
		err = errors.New("Invalid message data type")
	}
	return
}

// Event returns the message's event name.
func (m *Message) Event() string {
	return m.event
}

// Data returns the message event's data. 
func (m *Message) Data() map[string]interface{} {
	return m.data
}

// Returns value from the specified data field. If the key
// is not defined then returning nil.
func (m *Message) Get(key string) (value interface{}) {
	value, _ = m.data[key]
	return
}

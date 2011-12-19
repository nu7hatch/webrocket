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

import "testing"

func ValidateMessagePayload(msg *Message, t *testing.T) {
	if msg.Event() != "hello" {
		t.Errorf("Expected message event to be 'hello', given '%s'", msg.Event())
	}
	foo, ok := msg.Data()["foo"]
	if !ok || foo != "bar" {
		t.Errorf("Expected message data to contain {'foo': 'bar'}")
	}
}

func TestNewMessageFromJSON(t *testing.T) {
	_, err := newMessageFromJSON([]byte("invalid{"))
	if err == nil {
		t.Errorf("Expected error while parsing JSON")
	}
	msg, err := newMessageFromJSON([]byte("{\"hello\": {\"foo\": \"bar\"}}"))
	if err != nil {
		t.Errorf("Expected no error while creating message from JSON")
		return
	}
	ValidateMessagePayload(msg, t)
}

func TestNewMessage(t *testing.T) {
	_, err := newMessage(map[string]interface{}{})
	if err == nil || err.Error() != "Invalid message format" {
		t.Errorf("Expected error 'Invalid message format'")
	}
	_, err = newMessage(map[string]interface{}{"foo": "bar"})
	if err == nil || err.Error() != "Invalid message data type" {
		t.Errorf("Expected error 'Invalid message data type")
	}
	msg, err := newMessage(map[string]interface{}{"hello": map[string]interface{}{"foo": "bar"}})
	if err != nil {
		t.Errorf("Expected message to be created successfully")
		return
	}
	ValidateMessagePayload(msg, t)
}
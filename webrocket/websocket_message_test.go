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

func ValidateWebsocketMessagePayload(msg *WebsocketMessage, t *testing.T) {
	if msg.Event() != "hello" {
		t.Errorf("Expected message event to be 'hello', given '%s'", msg.Event())
	}
	foo, ok := msg.Data()["foo"]
	if !ok || foo != "bar" {
		t.Errorf("Expected message data to contain {'foo': 'bar'}")
	}
}

func TestNewWebsocketMessageFromJSON(t *testing.T) {
	_, err := newWebsocketMessageFromJSON([]byte("invalid{"))
	if err == nil {
		t.Errorf("Expected error while parsing JSON")
	}
	msg, err := newWebsocketMessageFromJSON([]byte("{\"hello\": {\"foo\": \"bar\"}}"))
	if err != nil {
		t.Errorf("Expected no error while creating message from JSON")
		return
	}
	ValidateWebsocketMessagePayload(msg, t)
}

func TestNewWebsocketMessage(t *testing.T) {
	_, err := newWebsocketMessage(map[string]interface{}{})
	if err == nil || err.Error() != "invalid message format" {
		t.Errorf("Expected error 'Invalid message format'")
	}
	_, err = newWebsocketMessage(map[string]interface{}{"foo": "bar"})
	if err == nil || err.Error() != "invalid message data type" {
		t.Errorf("Expected error 'Invalid message data type")
	}
	msg, err := newWebsocketMessage(map[string]interface{}{"hello": map[string]interface{}{"foo": "bar"}})
	if err != nil {
		t.Errorf("Expected message to be created successfully")
		return
	}
	ValidateWebsocketMessagePayload(msg, t)
}

func TestWebsocketMessageGet(t *testing.T) {
	msg, _ := newWebsocketMessage(map[string]interface{}{"hello": map[string]interface{}{"foo": "bar"}})
	if msg.Get("foo").(string) != "bar" {
		t.Errorf("Expected to get the value of the specified message's key")
	}
	if msg.Get("bar") != nil {
		t.Errorf("Expected to get nothing from not existing key")
	}
}

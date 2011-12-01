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
	"testing"
)

func TestNewMessageWithValidData(t *testing.T) {
	msg, err := NewMessage(map[string]interface{}{"hello":"world"})
	if err != nil {
		t.Errorf("Expected message to be ok, error found: %s", err.Error())
	}
	if msg.Event != "hello" {
		t.Errorf("Expected event to be `hello`, given %s", msg.Event)
	}
	if msg.Data.(string) != "world" {
		t.Errorf("Expected data to be `world`, given %s", msg.Data)
	}
}

func TestNewMessageWithInvalidData(t *testing.T) {
	_, err := NewMessage(map[string]interface{}{})
	if err == nil {
		t.Errorf("Expected message to be invalid")
	}
}
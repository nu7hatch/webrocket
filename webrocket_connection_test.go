// Copyright 2011 Chris Kowalik (nu7hatch). All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.
//
// Package webrocket implements advanced WebSocket server with custom
// protocols support. 
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
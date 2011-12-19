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

func TestNewChannel(t *testing.T) {
	ch, err := newChannel("hello")
	if err != nil {
		t.Errorf("Expected to create channel without errors")
	}
	if ch.Name() != "hello" {
		t.Errorf("Expected channel name to be 'hello', given '%s'", ch.Name())
	}
}

func TestNewChannelWithInvalidName(t *testing.T) {
	for _, name := range []string{".foo", "", "foo%", "-foo"} {
		_, err := newChannel(name)
		if err == nil || err.Error() != "Invalid name" {
			t.Errorf("Expected to throw 'Invalid name' error while creating a '%s' channel", name)
		}
	}
}

func TestChannelAddingAndRemovingSubscribers(t *testing.T) {
	ch, _ := newChannel("hello")
	c := newTestWebsocketClient()
	ch.addSubscriber(c)
	_, ok := ch.subscribers[c.Id()]
	if !ok {
		t.Errorf("Expected to add subscriber to the channel")
	}
	_, ok = c.subscriptions[ch.Name()]
	if !ok {
		t.Errorf("Expected to add subscription to the client")
	}
	ch.deleteSubscriber(c)
	_, ok = ch.subscribers[c.Id()]
	if ok {
		t.Errorf("Expected to remove subscriber from the channel")
	}
	_, ok = c.subscriptions[ch.Name()]
	if ok {
		t.Errorf("Expected to remove subscription from the client")
	}
}

func TestChannelSubscribersList(t *testing.T) {
	ch, _ := newChannel("hello")
	c := newTestWebsocketClient()
	ch.addSubscriber(c)
	if len(ch.Subscribers()) != 1 {
		t.Errorf("Expected subscribers list to contain one element")
	}
	if ch.Subscribers()[0].Id() != c.Id() {
		t.Errorf("Expected subscribers list to contain proper client")
	}
}
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
	ch, err := newChannel("hello", ChannelPresence)
	if err != nil {
		t.Errorf("Expected to create channel without errors")
	}
	if ch.Name() != "hello" {
		t.Errorf("Expected channel name to be 'hello', given '%s'", ch.Name())
	}
	if ch.Type() != ChannelPresence {
		t.Errorf("Expected channel to be presence one, given '%d'", ch.Type())
	}
}

func TestNewChannelWithInvalidName(t *testing.T) {
	for _, name := range []string{".foo", "", "foo%", "-foo"} {
		_, err := newChannel(name, ChannelNormal)
		if err == nil || err.Error() != "invalid channel name" {
			t.Errorf("Expected to throw 'invalid channel name' error while creating a '%s' channel", name)
		}
	}
}

func TestChannelIsPresence(t *testing.T) {
	ch, _ := newChannel("hello", ChannelPresence)
	if !ch.IsPresence() {
		t.Errorf("Expected to have a presence channel")
	}
	ch, _ = newChannel("hello", ChannelPrivate)
	if ch.IsPresence() {
		t.Errorf("Expected to not have a presence channel")
	}
	ch, _ = newChannel("hello", ChannelNormal)
	if ch.IsPresence() {
		t.Errorf("Expected to not have a presence channel")
	}
}

func TestChannelIsPrivate(t *testing.T) {
	ch, _ := newChannel("hello", ChannelPrivate)
	if !ch.IsPrivate() {
		t.Errorf("Expected to have a private channel")
	}
	ch, _ = newChannel("hello", ChannelPresence)
	if !ch.IsPrivate() {
		t.Errorf("Expected to have a private channel")
	}
	ch, _ = newChannel("hello", ChannelNormal)
	if ch.IsPrivate() {
		t.Errorf("Expected to not have a private channel")
	}
}

func TestChannelTypeFromName(t *testing.T) {
	ct := channelTypeFromName("presence-foobar")
	if ct != ChannelPresence {
		t.Errorf("Expected to have a presence channel type")
	}
	ct = channelTypeFromName("private-foobar")
	if ct != ChannelPrivate {
		t.Errorf("Expected to have a private channel type")
	}
	for _, name := range []string{"foobar", "presence", "private"} {
		ct = channelTypeFromName(name)
		if ct != ChannelNormal {
			t.Errorf("Expected to have a normal channel type")
		}
	}
}

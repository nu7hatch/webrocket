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

func TestNewStorageWithInvalidDir(t *testing.T) {
	s, err := newStorage("/dev/invalid/directory")
	if err == nil || s != nil {
		t.Errorf("Expected error while creating new storage")
	}
}

func TestNewStorage(t *testing.T) {
	s, err := newStorage("/tmp")
	if err != nil || s == nil {
		t.Errorf("Expected to create a storage without problems, error: %v", err)
	}
}

func TestStorageAddVhost(t *testing.T) {
	s, _ := newStorage("/tmp/")
	s.Clear()
	err := s.AddVhost("/test", "foo")
	if err != nil {
		t.Errorf("Expected to add vhost, error: %v", err)
	}
	vhosts, err := s.Vhosts()
	if err != nil {
		t.Errorf("Expected to get vhosts list, error: %v", err)
	}
	if len(vhosts) != 1 || vhosts[0].Name != "/test" || vhosts[0].AccessToken != "foo" {
		t.Errorf("Expected to add correct vhost")
	}
}

func TestStorageDeleteVhost(t *testing.T) {
	s, _ := newStorage("/tmp/")
	s.Clear()
	err := s.AddVhost("/test", "foo")
	if err != nil {
		t.Errorf("Expected to add vhost, error: %v", err)
	}
	err = s.DeleteVhost("/test")
	if err != nil {
		t.Errorf("Expected to delete vhost, error: %v", err)
	}
	vhosts, err := s.Vhosts()
	if err != nil {
		t.Errorf("Expected to get vhosts list, error: %v", err)
	}
	if len(vhosts) != 0 {
		t.Errorf("Expected to delete vhost")
	}
}

func TestStorageChangeVhostAccessToken(t *testing.T) {
	s, _ := newStorage("/tmp/")
	s.Clear()
	err := s.AddVhost("/test", "foo")
	if err != nil {
		t.Errorf("Expected to add vhost, error: %v", err)
	}
	s.AddVhost("/test", "bar")
	vhosts, _ := s.Vhosts()
	if len(vhosts) != 1 || vhosts[0].Name != "/test" || vhosts[0].AccessToken != "bar" {
		t.Errorf("Expected to change vhost's access token")
	}
}

func TestStorageAddChannel(t *testing.T) {
	s, _ := newStorage("/tmp/")
	s.Clear()
	err := s.AddVhost("/test", "foo")
	if err != nil {
		t.Errorf("Expected to add vhost, error: %v", err)
	}
	err = s.AddChannel("/test", "hello", ChannelPresence)
	if err != nil {
		t.Errorf("Expected to add channel, error: %v", err)
	}
	channels, err := s.Channels("/test")
	if err != nil {
		t.Errorf("Expected to get list of channels, %v", err)
	}
	if len(channels) != 1 || channels[0].Name != "hello" || channels[0].Kind != ChannelPresence {
		t.Errorf("Expected to add correct channel")
	}
}

func TestStorageDeleteChannel(t *testing.T) {
	s, _ := newStorage("/tmp/")
	s.Clear()
	err := s.AddVhost("/test", "foo")
	if err != nil {
		t.Errorf("Expected to add vhost, error: %v", err)
	}
	err = s.AddChannel("/test", "hello", ChannelPresence)
	if err != nil {
		t.Errorf("Expected to add channel, error: %v", err)
	}
	err = s.DeleteChannel("/test", "hello")
	if err != nil {
		t.Errorf("Expected to delete channel, error: %v", err)
	}
	channels, err := s.Channels("/test")
	if len(channels) != 0 {
		t.Errorf("Expected to delete channel")
	}
}

func TestStorageDeleteAllChannelsWhenVhostDeleted(t *testing.T) {
	s, _ := newStorage("/tmp/")
	s.Clear()
	err := s.AddVhost("/test", "foo")
	if err != nil {
		t.Errorf("Expected to add vhost, error: %v", err)
	}
	s.AddChannel("/test", "hello", ChannelPresence)
	s.AddChannel("/test", "world", ChannelPresence)
	err = s.DeleteVhost("/test")
	if err != nil {
		t.Errorf("Expected to delete vhost, error: %v", err)
	}
	channels, err := s.Channels("/test")
	if len(channels) != 0 {
		t.Errorf("Expected to delete all channels")
	}
}

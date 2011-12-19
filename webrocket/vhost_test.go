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

func newTestVhost() (v *Vhost, err error) {
	ctx := NewContext()
	v, err = newVhost(ctx, "/hello")
	return
}

func TestNewVhost(t *testing.T) {
	v, err := newTestVhost()
	if err != nil {
		t.Errorf("Expected to create a vhost, error encountered: %s", err.Error())
	}
	if v.Path() != "/hello" {
		t.Errorf("Expected vhost path to be '/hello', '%' given", v.Path())
	}
}

func TestNewVhostWithInvalidPath(t *testing.T) {
	ctx := NewContext()
	for _, path := range []string{"foo", "/", "/hello/", "/hello/foobar//"} {
		_, err := newVhost(ctx, path)
		if err == nil || err.Error() != "Invalid path" {
			t.Errorf("Expected to throw 'Invalid path' error while creating a vhost for '%s'", path)
		}
	}
}

func TestVhostGenerateSingleAccessToken(t *testing.T) {
	v, _ := newTestVhost()
	v.GenerateSingleAccessToken(".*")
	if len(v.permissions) != 1 {
		t.Errorf("Expected to generate single access token")
	}
}

func TestVhostValidateSingleAccessToken(t *testing.T) {
	v, _ := newTestVhost()
	token := v.GenerateSingleAccessToken(".*")
	pv, ok := v.ValidateSingleAccessToken(token)
	if !ok || pv.Token() != token {
		t.Errorf("Expected successfull validation of existing access token")
	}
	_, ok = v.ValidateSingleAccessToken(token)
	if ok {
		t.Errorf("Expected failure on double validation of the same access token")
	}
	_, ok = v.ValidateSingleAccessToken("foo")
	if ok {
		t.Errorf("Expected failed validation of not existing access token")
	}
}

func TestVhostOpenChannel(t *testing.T) {
	v, err := newTestVhost()
	ch, err := v.OpenChannel("hello")
	if err != nil || ch == nil {
		t.Errorf("Expected to create channel without errors")
	}
	vch, ok := v.channels["hello"]
	if !ok || vch == nil || vch.Name() != ch.Name() {
		t.Errorf("Expected to add channel to vhost's channels list")
	}
	_, err = v.OpenChannel("hello")
	if err == nil || err.Error() != "The 'hello' channel already exists" {
		t.Errorf("Expected error while creating duplicated channel")
	}
}

func TestVhostDeleteChannel(t *testing.T) {
	v, err := newTestVhost()
	v.OpenChannel("hello")
	err = v.DeleteChannel("hello")
	if err != nil {
		t.Errorf("Expected to delete channel without errors")
	}
	_, ok := v.channels["hello"]
	if ok {
		t.Errorf("Expected to unregister channel from vhost's channels list")
	}
	err = v.DeleteChannel("hello")
	if err == nil || err.Error() != "The 'hello' channel doesn't exist" {
		t.Errorf("Expected error while deleting non existing channel")
	}
}

func TestVhostGetChannel(t *testing.T) {
	v, err := newTestVhost()
	ch, _ := v.OpenChannel("hello")
	vch, err := v.Channel("hello")
	if err != nil || vch == nil || vch.Name() != ch.Name() {
		t.Errorf("Expected to get channel without errors")
	}
	_, err = v.Channel("john")
	if err == nil || err.Error() != "The 'john' channel doesn't exist" {
		t.Errorf("Expected to throw error while getting not existent channel")
	}
}

func TestVhostChannelsList(t *testing.T) {
	v, _ := newTestVhost()
	v.OpenChannel("hello")
	if len(v.Channels()) != 1 {
		t.Errorf("Expected the vhost's channels list to contain registered channel")
	}
}

// Copyright 2011 Chris Kowalik (nu7hatch). All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.
//
// Package webrocket implements advanced WebSocket server with custom
// protocols support. 
package webrocket

import (
	"testing"
	"log"
	"bytes"
)

func NewTestVhost() *Vhost {
	vhost := NewVhost("/foo")
	vhost.Log = log.New(bytes.NewBuffer([]byte{}), "a", log.LstdFlags)
	return vhost
}

func TestNewVhost(t *testing.T) {
	vhost := NewTestVhost()
	if vhost == nil {
		t.Errorf("Expected vhost to be ok, nil given")
	}
	if !vhost.IsRunning() {
		t.Errorf("Expected vhost to be running")
	}
}

func TestStopping(t *testing.T) {
	vhost := NewTestVhost()
	vhost.Stop()
	if vhost.IsRunning() {
		t.Errorf("Expected vhost to not be running")
	}
}

func TestAddUser(t *testing.T) {
	vhost := NewTestVhost()
	err := vhost.AddUser("foo", "bar", PermRead)
	if err != nil {
		t.Errorf("Expected to create new user")
	}
	user, ok := vhost.Users()["foo"]
	if !ok {
		t.Errorf("Expected to create new user")
	}
	if user.Name != "foo" {
		t.Errorf("Expected to create valid user")
	} 
}

func TestAddUserWithInvalidName(t *testing.T) {
	vhost := NewTestVhost()
	err := vhost.AddUser("", "bar", PermRead)
	if err == nil {
		t.Errorf("Expected to not create the invalid user")
	}
}

func TestAddUserWithInvalidPermission(t *testing.T) {
	vhost := NewTestVhost()
	err := vhost.AddUser("foo", "bar", 0)
	if err == nil {
		t.Errorf("Expected to not create the invalid user")
	}
}

func TestAddDuplicatedUser(t *testing.T) {
	vhost := NewTestVhost()
	vhost.AddUser("foo", "bar", PermRead)
	err := vhost.AddUser("foo", "bar", PermRead)
	if err == nil {
		t.Errorf("Expected to not create the duplicated user")
	}
}

func TestDeleteUser(t *testing.T) {
	vhost := NewTestVhost()
	vhost.AddUser("foo", "bar", PermRead)
	err := vhost.DeleteUser("foo")
	if err != nil {
		t.Errorf("Expected to delete user")
	}
	_, ok := vhost.Users()["foo"]
	if ok {
		t.Errorf("Expected to delete user")
	}
}

func TestDeleteNotExistingUser(t *testing.T) {
	vhost := NewTestVhost()
	err := vhost.DeleteUser("foo")
	if err == nil {
		t.Errorf("Expected to not delete not existing user")
	}
}

func TestSetUserPermissions(t *testing.T) {
	vhost := NewTestVhost()
	vhost.AddUser("foo", "bar", PermRead)
	user := vhost.Users()["foo"]
	if user.Permission != PermRead {
		t.Errorf("Expected to have only read permission")
	}
	err := vhost.SetUserPermissions("foo", PermRead|PermWrite)
	if err != nil {
		t.Errorf("Expected to set permissions without errors")
	}
	if user.Permission != PermRead|PermWrite {
		t.Errorf("Expected to have read and write permissions")
	}
}

func TestSetUserPermissionsWhenPermissionInvalid(t *testing.T) {
	vhost := NewTestVhost()
	vhost.AddUser("foo", "bar", PermRead)
	err := vhost.SetUserPermissions("foo", 0)
	if err == nil {
		t.Errorf("Expected to not set invalid permissions")
	}
}

func TestCreateChannel(t *testing.T) {
	vhost := NewTestVhost()
	ch := vhost.CreateChannel("bar")
	chans := vhost.Channels()
	_, ok := chans["bar"]
	if !ok {
		t.Errorf("Expected to open new channel")
	}
	if chans["bar"] != ch {
		t.Errorf("Expected to open new channel")
	}
}

func TestGetChannel(t *testing.T) {
	vhost := NewTestVhost()
	ch := vhost.CreateChannel("bar")
	cmp, _ := vhost.GetChannel("bar")
	if cmp != ch {
		t.Errorf("Expected to get proper channel")
	}
}

func TestGetChannelWhenNotExist(t *testing.T) {
	vhost := NewTestVhost()
	_, ok := vhost.GetChannel("bar")
	if ok {
		t.Errorf("Expected channel to not exist")
	}
}

func TestGetOrCreateChannel(t *testing.T) {
	vhost := NewTestVhost()
	if vhost.GetOrCreateChannel("bar") == nil {
		t.Errorf("Expected to autocreate channel")
	}
}
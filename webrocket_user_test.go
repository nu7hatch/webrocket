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

func TestCreate(t *testing.T) {
	user := NewUser("foo", "bar", 1)
	if user.Name != "foo" {
		t.Errorf("Expected user name to be `foo`, given %s", user.Name)
	}
	if user.Secret != "bar" {
		t.Errorf("Expected user secret to be `bar`, given %s", user.Secret)
	}
	if user.Permission != 1 {
		t.Errorf("Expected user permission to be `1`, given %s", user.Permission)
	}
}

func TestAuthenticateWhenSecretNotSpecified(t *testing.T) {
	user := NewUser("foo", "", 1)
	if !user.Authenticate("") {
		t.Errorf("Expected user to be authenticated")
	}
}

func TestAuthenticateWhenSecretSpecified(t *testing.T) {
	user := NewUser("foo", "bar", 1)
	if !user.Authenticate("bar") {
		t.Errorf("Expected user to be authenticated")
	}
	if user.Authenticate("foooo") {
		t.Errorf("Expected user to be not authenticated")
	}
}

func TestIsAllowedAndPermissions(t *testing.T) {
	user := NewUser("foo", "bar", PermRead)
	if !user.IsAllowed(PermRead) {
		t.Errorf("Expected user to be allowed to READ")
	}
	if user.IsAllowed(PermWrite) {
		t.Errorf("Expected user to not be allowed to WRITE")
	}
	if user.IsAllowed(PermManage) {
		t.Errorf("Expected user to not be allowed for MASTER role")
	}
	user = NewUser("foo", "bar", PermRead|PermWrite)
	if !user.IsAllowed(PermRead) {
		t.Errorf("Expected user to be allowed to READ")
	}
	if !user.IsAllowed(PermWrite) {
		t.Errorf("Expected user to be allowed to WRITE")
	}
	if user.IsAllowed(PermManage) {
		t.Errorf("Expected user to not be allowed for MASTER role")
	}
	user = NewUser("foo", "bar", PermRead|PermWrite|PermManage)
	if !user.IsAllowed(PermRead) {
		t.Errorf("Expected user to be allowed to READ")
	}
	if !user.IsAllowed(PermWrite) {
		t.Errorf("Expected user to be allowed to WRITE")
	}
	if !user.IsAllowed(PermManage) {
		t.Errorf("Expected user to be allowed for MASTER role")
	}
}
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

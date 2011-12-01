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

const (
	PermRead   = 1
	PermWrite  = 2
	PermManage = 4
)

// User keeps information about single configured user. 
type User struct {
	Name       string
	Secret     string
	Permission int
}

// Returns new, configured user.
func NewUser(name, secret string, permission int) *User {
	return &User{name, secret, permission}
}

// Authenticate matches given secret with current user credentials.
// If user's secret is empty, or given secret matches the defined one
// then returns true, otherwise returns false.
func (u *User) Authenticate(secret string) bool {
	return (u.Secret == "" || u.Secret == secret)
}

// IsAllowed checks if the user is permitted to do given operation.
func (u *User) IsAllowed(permission int) bool {
	return (u.Permission&permission == permission)
}

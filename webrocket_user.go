// Copyright 2011 Chris Kowalik (nu7hatch). All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.
//
// Package webrocket implements advanced WebSocket server with custom
// protocols support. 
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
	return (u.Permission & permission == permission)
}

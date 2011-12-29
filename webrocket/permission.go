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
	uuid "../uuid"
	"crypto/sha512"
	"fmt"
	"regexp"
)

// Generates single access token hash.
func generateSingleAccessToken() string {
	var hash = sha512.New()
	var uuid = uuid.GenerateTime()
	hash.Write([]byte(uuid))
	return fmt.Sprintf("%x", hash.Sum())
}

// Permission is represents single access token and permission
// pattern assigned to it. 
type Permission struct {
	pattern string
	token   string
}

// Creates new permission for specified pattern.
func NewPermission(pattern string) (p *Permission) {
	p = &Permission{pattern: pattern}
	p.token = generateSingleAccessToken()
	return p
}

// Checks if permission description allows to operate on specified
// channel. Speaking shortly just matches permission regexp with
// the channel name.
func (p *Permission) IsMatching(channel string) (ok bool) {
	re, err := regexp.Compile(fmt.Sprintf("^(%s)$", p.pattern))
	if err != nil {
		ok = false
		return
	}
	ok = re.MatchString(channel)
	return
}

// Token returns a string with the single access token hash.
func (p *Permission) Token() string {
	return p.token
}

// Pattern returns a permission regexp.
func (p *Permission) Pattern() string {
	return p.pattern
}

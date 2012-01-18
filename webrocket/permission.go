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
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"fmt"
	"regexp"
)

// Permission is represents single access token and permission pattern
// assigned to it. 
type Permission struct {
	// Permission regexp.
	pattern *regexp.Regexp
	// Generated unique single access token.
	token string
}

// Exported constructor
// -----------------------------------------------------------------------------

// Creates new permission for specified pattern.
//
// pattern - The regexp string to be used to match against the channels.
//
// Returns new permission or error if something went wrong.
func NewPermission(pattern string) (p *Permission, err error) {
	re, err := regexp.Compile(fmt.Sprintf("^(%s)$", pattern))
	if err != nil {
		err = errors.New("invalid permission regexp")
		return
	}
	p = &Permission{re, generateSingleAccessToken()}
	return
}

// Exported
// -----------------------------------------------------------------------------

// IsMatching checks if permission description allows to operate on the
// specified channel. Speaking shortly just matches permission regexp
// with the channel name.
//
// channel - The channel to be checked for permission.
//
// Examples:
//
//    p := NewPermission("(foo|bar)")
//    p.IsMatching("foo")
//    // => true
//    p.IsMatching("hello")
//    // => false
//
// Returns whether you have permission to operate on the channel or not.
func (p *Permission) IsMatching(channel string) bool {
	return p.pattern.MatchString(channel)
}

// Token returns single access token generate for this permission.
func (p *Permission) Token() string {
	return p.token
}

// Internal
// -----------------------------------------------------------------------------

// generateSingleAccessToken generates a single access token hash.
func generateSingleAccessToken() string {
	var buf [32]byte
	_, err := rand.Read(buf[:])
	if err != nil {
		return ""
	}
	hash := sha512.New()
	hash.Write(buf[:])
	return fmt.Sprintf("%x", hash.Sum([]byte{}))
}

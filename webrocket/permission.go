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
	"fmt"
	"regexp"
)

// Generates single access token hash.
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

// Permission is represents single access token and permission
// pattern assigned to it. 
type Permission struct {
	Pattern string
	Token   string
}

// Creates new permission for specified pattern.
func NewPermission(pattern string) *Permission {
	return &Permission{pattern, generateSingleAccessToken()}
}

// Checks if permission description allows to operate on specified
// channel. Speaking shortly just matches permission regexp with
// the channel name.
func (p *Permission) IsMatching(channel string) (ok bool) {
	re, err := regexp.Compile(fmt.Sprintf("^(%s)$", p.Pattern))
	if err != nil {
		ok = false
		return
	}
	ok = re.MatchString(channel)
	return
}
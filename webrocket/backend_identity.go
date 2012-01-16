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
	"errors"
	"regexp"
)

// A valid identity regexp.
const identityPattern = "^(dlr|req)\\:(/[\\w\\d\\-\\_]+(/[\\w\\d\\-\\_]+)*)\\:([\\d\\w]{40})\\:([\\d\\w\\-]{36})$"

// backendIdentity represents parsed identity information.
type backendIdentity struct {
	Raw         []byte
	Type        string
	AccessToken string
	Id          string
	Vhost       string
}

// Parses given identity string and returns its representation.
// The identity string has following format:
//
//     [type]:[vhost]:[access-token]:[client-id]
//
func parseBackendIdentity(raw []byte) (idty *backendIdentity, err error) {
	re, _ := regexp.Compile(identityPattern)
	parts := re.FindStringSubmatch(string(raw))
	if len(parts) != 6 {
		err = errors.New("Invalid identity")
		return
	}
	idty = &backendIdentity{
		Raw:         raw,
		Id:          parts[5],
		Vhost:       parts[2],
		AccessToken: parts[4],
		Type:        parts[1],
	}
	return
}

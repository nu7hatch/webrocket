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
	"strings"
)

// A valid identity regexp.
var backendIdentityPattern = regexp.MustCompile(
	"^(dlr|req)\\:(/[\\w\\d\\-\\_]+(/[\\w\\d\\-\\_]+)*)\\:([\\d\\w]{40})\\:([\\d\\w\\-]{36})$")

// backendIdentity represents a parsed identity information.
type backendIdentity struct {
	// The socket type.
	Type string
	// Related vhost.
	Vhost string
	// An access token used to authenticate.
	AccessToken string
	// Unique identifier of the client.
	Id string
}

// Internal
// -----------------------------------------------------------------------------

// parseBackendIdentity unpacks given identity string. The identity string
// has the following format:
//
//     [type]:[vhost]:[access-token]:[client-id]
//
// raw - The raw identity to be unpacked.
//
// Returns an unpacked identity or an error when something went wrong.
func parseBackendIdentity(raw string) (idty *backendIdentity, err error) {
	parts := backendIdentityPattern.FindStringSubmatch(raw)
	if len(parts) != 6 {
		err = errors.New("invalid identity")
		return
	}
	idty = &backendIdentity{
		Type:        parts[1],
		Vhost:       parts[2],
		AccessToken: parts[4],
		Id:          parts[5],
	}
	return
}

// Exported
// -----------------------------------------------------------------------------

// String returns the identity back in the string format.
func (idty *backendIdentity) String() string {
	return strings.Join([]string{
		idty.Type,
		idty.Vhost,
		idty.AccessToken,
		idty.Id,
	}, ";")
}

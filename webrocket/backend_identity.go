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
	zmq "../gozmq"
	"errors"
	"regexp"
)

// backendIdentity represents parsed identity information.
type backendIdentity struct {
	Raw         []byte
	Type        zmq.SocketType
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
	re, _ := regexp.Compile(
		"(dlr|req)\\:(/[\\w\\d\\-\\_]+(/[\\w\\d\\-\\_]+)*)\\:" +
			"([\\d\\w]{40})\\:([\\d\\w\\-]{36})")
	parts := re.FindStringSubmatch(string(raw))
	if len(parts) != 6 {
		err = errors.New("Invalid identity")
		return
	}
	idty = &backendIdentity{Raw: raw}
	if parts[1] == "dlr" {
		idty.Type = zmq.DEALER
	} else {
		idty.Type = zmq.REQ
	}
	idty.Vhost = parts[2]
	idty.AccessToken = parts[4]
	idty.Id = parts[5]
	return
}

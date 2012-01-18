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
	"bytes"
	"errors"
	"fmt"
)

// backendRequest implements a structure which represents a single message
// incoming from the backend client.
type backendRequest struct {
	// The request owner's connection.
	conn *backendConnection
	// The connection's identity string.
	Identity string
	// The command name.
	Command string
	// The rest of the message.
	Message [][]byte
}

// Internal constructor
// -----------------------------------------------------------------------------

// newBackendRequest creates a new backend request object and returns it.
//
// conn - The request owner's connection.
// id   - The identity line extracted from the payload.
// cmd  - The command line extracted from the payload.
// msg  - The rest of the message.
//
func newBackendRequest(conn *backendConnection, id []byte, cmd []byte,
	msg [][]byte) (r *backendRequest) {
	return &backendRequest{
		conn:     conn,
		Identity: string(id),
		Command:  string(cmd),
		Message:  msg,
	}
}

// Exported
// -----------------------------------------------------------------------------

// Reply sends a specified response to the request owner's connection.
// It kills the connection aftrer the message is sent.
//
// cmd    - The command to be send.
// frames - The other parts of the message.
//
// Returns an error if something went wrong.
func (r *backendRequest) Reply(cmd string, frames ...string) (err error) {
	if r == nil || r.conn == nil {
		err = errors.New("broken connection")
		return
	}
	err = r.conn.Send(cmd, frames...)
	r.conn.Kill()
	return err
}

// String converts the message to readable format.
func (r *backendRequest) String() string {
	s := string(bytes.Join(r.Message, []byte(", ")))
	return fmt.Sprintf("[%s, %s]", r.Command, s)
}

// Len returns a number of message's frames.
func (r *backendRequest) Len() int {
	return len(r.Message)
}

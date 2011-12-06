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

import "errors"

// message is a simple structure which keeps the incoming events
// information and data.
type Message struct {
	Event string
	Data  interface{}
}

// extractMessage converts received map into message structure. 
func NewMessage(data map[string]interface{}) (*Message, error) {
	if len(data) != 1 {
		return nil, errors.New("Invalid message format")
	}
	msg := &Message{}
	for k := range data {
		msg.Event = k
	}
	msg.Data = data[msg.Event]
	return msg, nil
}

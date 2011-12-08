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

// Creates new error payload.
func newError(id string) map[string]interface{} {
	return map[string]interface{}{"id": id}
}

// Predefined error payloads.
var (
	ErrInvalidDataReceived  = newError("INVALID_DATA_RECEIVED")
	ErrInvalidMessageFormat = newError("INVALID_MESSAGE_FORMAT")
	ErrInvalidPayload       = newError("INVALID_PAYLOAD")
	ErrAccessDenied         = newError("ACCESS_DENIED")
	ErrInvalidUserName      = newError("INVALID_USER_NAME")
	ErrUserNotFound         = newError("USER_NOT_FOUND")
	ErrInvalidCredentials   = newError("INVALID_CREDENTIALS")
	ErrInvalidChannelName   = newError("INVALID_CHANNEL_NAME")
	ErrChannelNotFound      = newError("CHANNEL_NOT_FOUND")
	ErrInvalidEventName     = newError("INVALID_EVENT_NAME")
	ErrUndefinedEvent       = newError("UNDEFINED_EVENT")
)

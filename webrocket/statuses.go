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

// Log messages.
var logMsg = map[string]string{
	"200": "%s[%s]: {%s} CONNECTED sid=`%s`",
	"201": "%s[%s]: {%s} AUTHENTICATED access=`%s` sid=`%s`",
	"202": "%s[%s]: {%s} SUBSCRIBED channel=`%s` sid=`%s`",
	"203": "%s[%s]: {%s} UNSUBSCRIBED channel=`%s` sid=`%s`",
	"204": "%s[%s]: {%s} BROADCASTED event=`%s` channel=`%s` sid=`%s`",
	"205": "%s[%s]: {%s} TRIGGERED event=`%s` sid=`%s`",
	"207": "%s[%s]: {%s} CLOSED sid=`%s`",
	"250": "%s[%s]: {%s} CHANNEL_OPENED channel=`%s`",
	"251": "%s[%s]: {%s} CHANNEL_EXISTS channel=`%s`",
	"252": "%s[%s]: {%s} CHANNEL_CLOSED channel=`%s`",
	"253": "%s[%s]: {%s} SINGLE_ACCESS_TOKEN_GENERATED permission=`%s`",
	"400": "%s[%s]: {%s} BAD_REQUEST error=`%s`",
	"402": "%s[%s]: {%s} UNAUTHORIZED error=`%s`",
	"403": "%s[%s]: {%s} FORBIDDEN channel=`%s`",
	"451": "%s[%s]: {%s} INVALID_CHANNEL_NAME channel=`%s`",
	"453": "%s[%s]: {%s} NOT_SUBSCIRBED channel=`%s`",
	"454": "%s[%s]: {%s} CHANNEL_NOT_FOUND channel=`%s`",
	"500": "%s[%s]: {%s} INTERNAL_ERROR error=`%s`",
	"597": "%s[%s]: {%s} COULD_NOT_SEND error=`%s`",
	"598": "%s[%s]: {%s} END_OF_FILE error=`%s`",
}

// Possible success payloads (backend only).
var (
	okChannelOpened = okEvent(250, "Channel opened")
	okChannelExists = okEvent(251, "Channel exists")
	okChannelClosed = okEvent(252, "Channel closed")
	okBroadcasted   = okEvent(204, "Broadcasted")
)

// Possible error payloads.
var (
	errorBadRequest         = errorEvent(400, "Bad request")
	errorUnauthorized       = errorEvent(402, "Unauthorized")
	errorForbidden          = errorEvent(403, "Forbidden")
	errorInvalidChannelName = errorEvent(452, "Invalid channel name")
	errorInvalidEventName   = errorEvent(452, "Invalid event name")
	errorNotSubscribed      = errorEvent(453, "Not subscribed")
	errorChannelNotFound    = errorEvent(454, "Channel not found")
	errorInternal           = errorEvent(500, "Internal error")
)

// Returns an error event's payload.
func errorEvent(code int, status string) map[string]interface{} {
	return map[string]interface{}{
		"__error": map[string]interface{}{
			"code":   code,
			"status": status,
		},
	}
}

// Returns a success event's payload.
func okEvent(code int, status string) map[string]interface{} {
	return map[string]interface{}{
		"__ok": map[string]interface{}{
			"code":   code,
			"status": status,
		},
	}
}

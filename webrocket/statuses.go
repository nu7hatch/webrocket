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

// List of possible status codes returned by WebRocket:
//
// = Success codes
//
// * 201: Authenticated
// * 202: Subscribed
// * 203: Unsubscribed
// * 204: Broadcasted
// * 205: Triggered
// * 207: Closed
// * 250: Channel opened
// * 251: Channel exists // TODO: rename to 350
// * 252: Channel closed
// * 270: Single access token generated
//
// = Error codes
//
// * 400: Bad request
// * 402: Unauthorized
// * 403: Forbidden
// * 451: Invalid channel name
// * 453: Not subscribed
// * 454: Channel not found
// * 597: Internal error
// * 598: End of file
//
// = Information and error codes (backend only)
//
// * 300: Ready
// * 301: Heartbeat
// * 305: Connected
// * 408: Expired
//

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

// Subscription represents single subscription entry.
type Subscription struct {
	// The subscriber's connection.
	client *WebsocketConnection
	// Whether this subscriber is hidden or not.
	hidden bool
	// Data attached to this subscription (used only by the presence channels).
	data map[string]interface{}
}

// Internal constructor
// -----------------------------------------------------------------------------

// newSubscription creates new Subscription object for given connection.
// If hidden option is true, then this subscription will be invisible for
// the other subscribers of the presence channel.
func newSubscription(c *WebsocketConnection, hidden bool, data map[string]interface{}) *Subscription {
	return &Subscription{c, hidden, data}
}

// Exported
// -----------------------------------------------------------------------------

// Client returns the subscribers's connection. 
func (s *Subscription) Client() *WebsocketConnection {
	return s.client
}

// IsHidden returns whether this subscription is hidden or not.
func (s *Subscription) IsHidden() bool {
	return s.hidden
}

// Data returns a data attached to this subscription.
func (s *Subscription) Data() map[string]interface{} {
	return s.data
}

// Id returns an unique id of the subscriber's connection.
func (s *Subscription) Id() (id string) {
	if s.client != nil {
		id = s.Id()
	}
	return
}

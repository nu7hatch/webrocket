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

// subscription struct is used to modify channel subscription state
// from within the handler.
type subscription struct {
	conn   *conn
	active bool
}

// channel keeps information about specified channel and it's subscriptions.
// It's hub is used to broadcast messages.
type Channel struct {
	name        string
	vhost       *Vhost
	subscribers map[*conn]bool
	subscribe   chan subscription
	broadcast   chan interface{}
}

// NewChannel creates and configures new channel in specified vhost.
func NewChannel(v *Vhost, name string) *Channel {
	ch := &Channel{name: name, vhost: v, subscribers: make(map[*conn]bool)}
	ch.subscribe, ch.broadcast = make(chan subscription), make(chan interface{})
	go ch.hub()
	return ch
}

// Channel's hub manages subscriptions and broacdasts messages to all subscribers.
func (ch *Channel) hub() {
	for {
		select {
		case s := <-ch.subscribe:
			if s.active {
				ch.subscribers[s.conn] = true
				s.conn.channels[ch] = true
			} else {
				delete(ch.subscribers, s.conn)
				delete(s.conn.channels, ch)
			}
		case payload := <-ch.broadcast:
			for conn := range ch.subscribers {
				conn.send(payload)
			}
		}
	}
}

// Returns name of the channel. 
func (ch *Channel) Name() string {
	return ch.name
}

// Returns list of subscribers.
func (ch *Channel) Subscribers() []*conn {
	conns, i := make([]*conn, len(ch.subscribers)), 0
	for conn, _ := range ch.subscribers {
		conns[i] = conn
		i += 1
	}
	return conns
}
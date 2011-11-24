// Copyright 2011 Chris Kowalik (nu7hatch). All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.
//
// Package webrocket implements advanced WebSocket server with custom
// protocols support. 
package webrocket

// subscription struct is used to modify channel subscription state
// from within the handler.
type subscription struct {
	conn   *conn
	active bool
}

// channel keeps information about specified channel and it's subscriptions.
// It's hub is used to broadcast messages.
type channel struct {
	name        string
	vhost       *Vhost
	subscribers map[*conn]bool
	subscribe   chan subscription
	broadcast   chan interface{}
}

// newChannel creates and configures new channel in specified vhost.
func newChannel(v *Vhost, name string) *channel {
	ch := &channel{name: name, vhost: v, subscribers: make(map[*conn]bool)}
	ch.subscribe, ch.broadcast = make(chan subscription), make(chan interface{})
	go ch.hub()
	return ch
}

// Channel's hub manages subscriptions and broacdasts messages to all subscribers.
func (ch *channel) hub() {
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

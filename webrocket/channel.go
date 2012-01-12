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
	"sync"
)

// Pattern used to validate channel name.
const channelNamePattern = "^[\\w\\d\\_][\\w\\d\\-\\_\\.]*$"

// Subscription entry.
type subscription struct {
	c  *WebsocketClient
	ok bool
}

// Channel keeps information about specified channel and it's subscriptions.
// It's hub is used to broadcast messages.
type Channel struct {
	name        string
	subscribers map[string]*WebsocketClient
	broadcast   chan interface{}
	mtx         sync.Mutex
}

// NewChannel creates and configures new channel in specified vhost.
// The channel name limitations are the same as in case of user names,
// can be composed with letters, numbers, dashes, underscores and dots.
//
// Each channel's broadcaster works in separate goroutine.
func newChannel(name string) (ch *Channel, err error) {
	re, _ := regexp.Compile(channelNamePattern)
	if !re.MatchString(name) {
		err = errors.New("invalid name")
		return
	}
	ch = &Channel{
		name:        name,
		subscribers: make(map[string]*WebsocketClient),
		broadcast:   make(chan interface{}),
	}
	go ch.broadcastLoop()
	return
}

// broadcastLoop runs a broadcaster's event loop.
func (ch *Channel) broadcastLoop() {
	for payload := range ch.broadcast {
		for _, client := range ch.subscribers {
			client.Send(payload)
		}
	}
	for _, c := range ch.subscribers {
		ch.deleteSubscriber(c)
	}
}

// hasSubscriber checks if specified client is subscribing to
// this channel.
func (ch *Channel) hasSubscriber(client *WebsocketClient) bool {
	if client != nil {
		_, ok := ch.subscribers[client.Id()]
		return ok
	}
	return false
}

// addSubscriber adds given subscription to this channel.
func (ch *Channel) addSubscriber(client *WebsocketClient) {
	if client != nil {
		ch.subscribers[client.Id()] = client
		client.subscriptions[ch.Name()] = ch
	}
}

// deleteSubscriber removes given subscription from this channel.
func (ch *Channel) deleteSubscriber(client *WebsocketClient) {
	if client != nil {
		delete(ch.subscribers, client.Id())
		delete(client.subscriptions, ch.Name())
	}
}

// Returns name of the channel. 
func (ch *Channel) Name() string {
	return ch.name
}

// If ok is true, then adds subsriber, otherwise removes it from
// the channel.
func (ch *Channel) Subscribe(client *WebsocketClient, ok bool) {
	if !ch.IsAlive() {
		return
	}
	ch.mtx.Lock()
	defer ch.mtx.Unlock()
	if ok {
		ch.addSubscriber(client)
	} else {
		ch.deleteSubscriber(client)
	}
}

// Broadcast sends specified payload to all active subscribers
// of this channel.
func (ch *Channel) Broadcast(payload interface{}) {
	if ch.IsAlive() {
		ch.broadcast <- payload
	}
}

// Returns list of clients subsribing to this channel.
func (ch *Channel) Subscribers() (clients []*WebsocketClient) {
	clients, i := make([]*WebsocketClient, len(ch.subscribers)), 0
	for _, client := range ch.subscribers {
		clients[i] = client
		i += 1
	}
	return
}

// Returns true if the channel is active.
func (ch *Channel) IsAlive() bool {
	ch.mtx.Lock()
	defer ch.mtx.Unlock()
	return ch.broadcast != nil
}

// Stops execution of the channel.
func (ch *Channel) Kill() {
	ch.mtx.Lock()
	defer ch.mtx.Unlock()
	close(ch.broadcast)
	ch.broadcast = nil
}

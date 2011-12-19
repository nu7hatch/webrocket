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
	"errors"
	"sync"
	"regexp"
)

// Pattern used to validate channel name.
const channelNamePattern = "^[\\w\\d\\_][\\w\\d\\-\\_\\.]*$"

// Channel keeps information about specified channel and it's subscriptions.
// It's hub is used to broadcast messages.
type Channel struct {
	name        string
	subscribers map[string]*WebsocketClient
	mtx         sync.Mutex
	isRunning   bool
}

// NewChannel creates and configures new channel in specified vhost.
// The channel name limitations are the same as in case of user names,
// can be composed with letters, numbers, dashes, underscores and dots.
func newChannel(name string) (ch *Channel, err error) {
	re, _ := regexp.Compile(channelNamePattern)
	if !re.MatchString(name) {
		err = errors.New("Invalid name")
		return
	}
	ch = &Channel{name: name, isRunning: true}
	ch.subscribers = make(map[string]*WebsocketClient)
	return
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
		ch.mtx.Lock()
		defer ch.mtx.Unlock()
		ch.subscribers[client.Id()] = client
		client.subscriptions[ch.Name()] = ch
	}
}

// deleteSubscriber removes given subscription from this channel.
func (ch *Channel) deleteSubscriber(client *WebsocketClient) {
	if client != nil {
		ch.mtx.Lock()
		defer ch.mtx.Unlock()
		delete(ch.subscribers, client.Id())
		delete(client.subscriptions, ch.Name())
	}
}

// Broadcast sends specified payload to all active subscribers
// of this channel.
func (ch *Channel) Broadcast(payload interface{}) {
	if !ch.isRunning {
		return
	}
	go func() {
		for _, client := range ch.subscribers {
			if client != nil {
				client.Send(payload)
			}
		}
	}()
}

// Returns name of the channel. 
func (ch *Channel) Name() string {
	return ch.name
}

// Returns list of clients subsribing to this channel.
func (ch *Channel) Subscribers() (clients []*WebsocketClient) {
	i := 0
	clients = make([]*WebsocketClient, len(ch.subscribers))
	for _, client := range ch.subscribers {
		clients[i] = client
		i += 1
	}
	return
}

// Stops execution of the channel.
func (ch *Channel) kill() {
	ch.mtx.Lock()
	defer ch.mtx.Unlock()
	ch.isRunning = false
}
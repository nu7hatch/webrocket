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
	"strings"
)

// Pattern used to validate a channel name.
var validChannelNamePattern = regexp.MustCompile("^[\\w\\d\\_][\\w\\d\\-\\_\\.]*$")

// Internal types
// -----------------------------------------------------------------------------

// payload implements a structure used to broadcast information on the
// channels. If includeHidden is false, then message will not be send
// to hidden subscribers. 
type payload struct {
	// Send to hidden subscribers as well.
	includeHidden bool
	// Data's payload.
	payload map[string]interface{}
}

// Exported types
// -----------------------------------------------------------------------------

// ChannelType represents a type of the channel. Can be normal, private
// or presence.
type ChannelType int

// Possible channel types
const (
	ChannelNormal   = 0
	ChannelPrivate  = 2
	ChannelPresence = 3
)

// Channel keeps information about specified channel and it's subscriptions.
// It's hub is used to broadcast messages.
type Channel struct {
	// The name of the channel.
	name string
	// A type of the channel.
	kind ChannelType
	// List of subscribers.
	subscribers map[string]*Subscription
	// Broadcasting channel.
	broadcast chan *payload
	// Internal semaphore.
	mtx sync.Mutex
}

// Internal constructor
// -----------------------------------------------------------------------------

// newChannel creates and configures new channel under the specified vhost.
// The channel name limitations are the same as in case of user names: channel
// name can be contain letters, numbers, dashes, underscores and dots.
//
// Each channel's broadcasting loop works in its own goroutine.
//
// name - The name of a new channel.
// kind - The type of a new channel.
//
// Returns new channel or error if something went wrong.
func newChannel(name string, kind ChannelType) (ch *Channel, err error) {
	if !validChannelNamePattern.MatchString(name) {
		err = errors.New("invalid channel name")
		return
	}
	ch = &Channel{
		name:        name,
		kind:        kind,
		subscribers: make(map[string]*Subscription),
		broadcast:   make(chan *payload),
	}
	go ch.broadcastLoop()
	return
}

// Internal
// -----------------------------------------------------------------------------

// channelTypeFromName parses given channel name and discover's what is
// its type.
//
// name - The name to be parsed.
//
// Returns the channel type.
func channelTypeFromName(name string) (t ChannelType) {
	if parts := strings.Split(name, "-"); len(parts) > 1 {
		switch parts[0] {
		case "presence":
			return ChannelPresence
		case "private":
			return ChannelPrivate
		}
	}
	return ChannelNormal
}

// broadcastLoop runs a broadcaster's event loop.
func (ch *Channel) broadcastLoop() {
	for x := range ch.broadcast {
		for _, s := range ch.subscribers {
			if s.IsHidden() && !x.includeHidden {
				// Skip hidden subscriber.
				continue
			}
			if client := s.Client(); client != nil {
				client.Send(x.payload)
			}
		}
	}
	// Delete all subscribers after the channel is killed. 
	for _, s := range ch.subscribers {
		ch.unsubscribe(s.Client(), map[string]interface{}{})
	}
}

// subscribe appends given client to the list of subscribers. If hidden
// is true then he will be invisible fot the other subscribers of the
// presence channel. Threadsafe, May be called from many websocket
// connection's handlers.
//
// client - The websocket client to be subscribed.
// hidden - If true then subscription will be invisible.
// data   - The user specific data attached to the presence channel identity.
//
func (ch *Channel) subscribe(client *WebsocketConnection, hidden bool, data map[string]interface{}) {
	if client != nil && ch.IsAlive() {
		ch.mtx.Lock()
		sid := client.Id()
		if _, ok := ch.subscribers[sid]; ok {
			// Already subscribing this channel...
			ch.mtx.Unlock()
			return
		}
		ch.subscribers[sid] = newSubscription(client, hidden, data)
		client.subscriptions[ch.Name()] = ch
		if ch.IsPresence() && !hidden {
			// Tell the others that someone joined the channel.
			ch.broadcast <- &payload{true,
				map[string]interface{}{
					"__memberJoined": map[string]interface{}{
						"sid":     sid,
						"channel": ch.name,
						"data":    data,
					},
				},
			}
		}
		ch.mtx.Unlock()
		// Confirm subscription.
		client.Send(map[string]interface{}{
			"__subscribed": map[string]interface{}{
				"channel": ch.name,
				// TODO: add subscribers list if channel is a presence one...
			},
		})
	}
}

// unsubscribe removes specified client from the subscribers list. Threadsafe,
// may be called from many websocket connection's handlers.
//
// client - The websocket client to be subscribed.
// data   - The user specific data passed to other subscribers.
//
func (ch *Channel) unsubscribe(client *WebsocketConnection, data map[string]interface{}) {
	if client != nil && ch.IsAlive() {
		var s *Subscription
		var ok bool
		ch.mtx.Lock()
		sid := client.Id()
		if s, ok = ch.subscribers[sid]; !ok {
			ch.mtx.Unlock()
			return
		}
		delete(ch.subscribers, sid)
		delete(client.subscriptions, ch.Name())
		if ch.IsPresence() && !s.IsHidden() {
			// Tell the others that this guy is not subscribing the
			// channel anymore.
			ch.broadcast <- &payload{true,
				map[string]interface{}{
					"__memberLeft": map[string]interface{}{
						"sid":     sid,
						"channel": ch.name,
						"data":    data,
					},
				},
			}
		}
		ch.mtx.Unlock()
		// Confirm unsubscription.
		client.Send(map[string]interface{}{
			"__unsubscribed": map[string]interface{}{
				"channel": ch.name,
			},
		})
	}
}

// Exported
// -----------------------------------------------------------------------------

// Name returns name of the channel. 
func (ch *Channel) Name() string {
	return ch.name
}

// Type returns flag representing the channel's type.
func (ch *Channel) Type() ChannelType {
	return ch.kind
}

// IsPrivate returns whether this channel requires authenticaion or not.
func (ch *Channel) IsPrivate() bool {
	return ch.kind&ChannelPrivate == ChannelPrivate
}

// IsPresence returns whether this channel is a presence one or not.
func (ch *Channel) IsPresence() bool {
	return ch.kind&ChannelPresence == ChannelPresence
}

// HasSubscriber checks whether specified client is subscribing to this
// channel or not. Threadsafe, May be called from many places and depends
// on the Subscribe and Unsubscribe funcs.
//
// client - The websocket client to be checked.
//
// Returns whether client is subscribing this channel or not.
func (ch *Channel) HasSubscriber(client *WebsocketConnection) bool {
	if client != nil {
		ch.mtx.Lock()
		defer ch.mtx.Unlock()
		_, ok := ch.subscribers[client.Id()]
		return ok
	}
	return false
}

// Subscribers returns list of the clients subsribing to the channel.
// Threadsafe, May be called from many places and depends on the Subscribe
// and Unsubscribe funcs.
func (ch *Channel) Subscribers() map[string]*Subscription {
	ch.mtx.Lock()
	defer ch.mtx.Unlock()
	return ch.subscribers
}

// Broadcast sends given payload to all active subscribers of this channel.
// Threadsafe, May be called from many websocket client's handlers.
//
// x - The data to be broadcasted to all the subscribers.
//
func (ch *Channel) Broadcast(x map[string]interface{}) {
	if ch.IsAlive() {
		ch.broadcast <- &payload{false, x}
	}
}

// IsAlive returns whether the channels is alive or not. Threadsafe, May be
// called from many places and depends on the Kill func.
func (ch *Channel) IsAlive() bool {
	ch.mtx.Lock()
	defer ch.mtx.Unlock()
	return ch.broadcast != nil
}

// Kill closes the channel's broadcaster and marks it as dead. Threadsafe, 
// May be called from the backend protocol or admin interface and
// affects the IsAlive func.
func (ch *Channel) Kill() {
	ch.mtx.Lock()
	defer ch.mtx.Unlock()
	if ch.broadcast != nil {
		close(ch.broadcast)
		ch.broadcast = nil
	}
}

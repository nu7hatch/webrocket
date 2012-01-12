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
	"container/ring"
	"sync"
	"time"
)

const (
	backendLobbyDefaultMaxRetries = 3
	backendLobbyDefaultRetryDelay = 2e6 * time.Nanosecond
)

// backendLobby coordinates messages flow between the WebRocket and all
// connected backend application agents.  
type backendLobby struct {
	agents     map[string]*BackendAgent
	queue      chan interface{}
	dead       chan bool
	robin      *ring.Ring
	maxRetries int
	retryDelay time.Duration
	mtx        sync.Mutex
}

// Creates new backendLobby exchange.
func newBackendLobby() (l *backendLobby) {
	l = &backendLobby{
		robin:      nil,
		agents:     make(map[string]*BackendAgent),
		queue:      make(chan interface{}),
		dead:       make(chan bool),
		maxRetries: backendLobbyDefaultMaxRetries,
		retryDelay: backendLobbyDefaultRetryDelay,
	}
	go l.dequeue()
	return l
}

// Enqueues given message.
func (l *backendLobby) enqueue(payload interface{}) {
	l.queue <- payload
}

// dequeue is an event loop which picks enqueued message and
// load ballances it across connected agents.
func (l *backendLobby) dequeue() {
	for payload := range l.queue {
		l.send(payload)
	}
	for _, agent := range l.agents {
		agent.Kill()
	}
}

// Dequeues single message and load ballances it across connected
// clients.
func (l *backendLobby) send(payload interface{}) {
	retries := 0
start:
	agent := l.loadBallance()
	if agent != nil {
		agent.Trigger(payload)
		return
	}
	// No agents available, waiting a while and retrying
	// TODO: log error
	if retries >= l.maxRetries {
		<-time.After(l.retryDelay)
		retries += 1
		goto start
	}
}

// Adds specified agent to the load ballancer ring using the round
// robin method.
func (l *backendLobby) addAgent(agent *BackendAgent) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.agents[string(agent.id)] = agent
	r := ring.New(1)
	r.Value = agent
	if l.robin == nil {
		l.robin = r
	} else {
		l.robin.Link(r)
	}
}

// Removes specified agent from the load ballancer ring.
func (l *backendLobby) deleteAgent(agent *BackendAgent) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	delete(l.agents, string(agent.id))
}

// Find agent with specified ID and returns it.
func (l *backendLobby) getAgentById(id string) (agent *BackendAgent, ok bool) {
	agent, ok = l.agents[id]
	return
}

// Moves agent cursor to the next available one. 
func (l *backendLobby) loadBallance() (agent *BackendAgent) {
start:
	if l.robin == nil || l.robin.Len() == 0 {
		return
	}
	next := l.robin.Next()
	agent, ok := next.Value.(*BackendAgent)
	if !ok {
		return
	}
	// Now we're checking if the agent wasn't deleted
	_, ok = l.agents[string(agent.id)]
	if !ok {
		// If it was deleted, we have to remove it from the
		// ring as well.
		l.robin.Unlink(1)
		agent = nil
		goto start
	} else {
		l.robin = next
	}
	return
}

// Returns true if lobby is running.
func (l *backendLobby) IsAlive() bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	return l.queue != nil
}

// Stops execution of this lobby.
func (l *backendLobby) Kill() {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	close(l.queue)
	l.queue = nil
}

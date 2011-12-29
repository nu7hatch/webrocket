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
	"container/ring"
	"sync"
	"time"
)

const (
	backendLobbyDefaultMaxRetries = 3
	backendLobbyDefaultRetryDelay = time.Duration(2e6)
)

// backendLobby coordinates messages flow between the WebRocket and all
// connected backend application agents.  
type backendLobby struct {
	agents     map[string]*BackendAgent
	queue      chan interface{}
	robin      *ring.Ring
	maxRetries int
	retryDelay time.Duration
	isRunning  bool
	mtx        sync.Mutex
}

// Creates new backendLobby exchange.
func newBackendLobby() (l *backendLobby) {
	l = new(backendLobby)
	l.isRunning = true
	l.robin = nil
	l.agents = make(map[string]*BackendAgent)
	l.queue = make(chan interface{})
	l.maxRetries = backendLobbyDefaultMaxRetries
	l.retryDelay = backendLobbyDefaultRetryDelay
	go l.dequeue()
	return l
}

// Enqueues given message.
func (l *backendLobby) enqueue(payload interface{}) {
	if l.isRunning {
		l.queue <- payload
	}
}

// dequeue is an event loop which picks enqueued message and
// load ballances it across connected agents.
func (l *backendLobby) dequeue() {
	defer l.kill()
	for {
		if !l.isRunning {
			break
		}
		l.doDequeue()
	}
}

// Dequeues single message and load ballances it across connected
// clients.
func (l *backendLobby) doDequeue() {
	payload := <-l.queue
	retries := 0
	for {
		if retries > l.maxRetries {
			// Retries limit reached, dropping the message...
			break
		}
		agent := l.roundRobin()
		if agent == nil {
			// No agents available, waiting a while and retrying
			time.Sleep(l.retryDelay)
			retries += 1
			// TODO: error, no clients available
			continue
		}
		agent.Send(payload)
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
func (l *backendLobby) roundRobin() (agent *BackendAgent) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

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

// Stops execution of this lobby.
func (l *backendLobby) kill() {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.isRunning = false
}

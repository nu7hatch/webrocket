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

// Backend bobby defaults.
const (
	backendLobbyDefaultMaxRetries = 3
	backendLobbyDefaultRetryDelay = 2e6 * time.Nanosecond
)

// backendLobby coordinates messages flow between the WebRocket and all
// connected backend application workers.  
type backendLobby struct {
	// List of active workers.
	workers map[string]*BackendWorker
	// Internal queue.
	queue chan interface{}
	// The load ballancing ring.
	robin *ring.Ring
	// The maximum number of retries to send a message. 
	maxRetries int
	// The delay time before the next try to send a message.
	retryDelay time.Duration
	// Internal semaphore.
	mtx sync.Mutex
}

// Internal constructor
// -----------------------------------------------------------------------------

// newBackendLobby creates new backend lobby exchange and initializes
// a goroutine for dequeueing and sending the messages.
//
// Returns new backend lobby object.
func newBackendLobby() (l *backendLobby) {
	l = &backendLobby{
		robin:      nil,
		workers:    make(map[string]*BackendWorker),
		queue:      make(chan interface{}),
		maxRetries: backendLobbyDefaultMaxRetries,
		retryDelay: backendLobbyDefaultRetryDelay,
	}
	go l.dequeueLoop()
	return l
}

// Internal
// -----------------------------------------------------------------------------

// dequeueLoop is an event loop which waits for the messages and load ballances
// it across all the connected workers.
func (l *backendLobby) dequeueLoop() {
	for payload := range l.queue {
		l.send(payload)
	}
	// We have to kill all the workers when it terminates... 
	for _, worker := range l.workers {
		worker.Kill()
	}
}

// send requests for a worker from the load ballancer and sends given data
// to it. If it's not possible to get free worker then it tries again
// until the maxRetries limit is reached.
//
// payload - The data to be send.
//
func (l *backendLobby) send(payload interface{}) {
	retries := 0
start:
	if worker := l.getAvailableWorker(); worker != nil {
		worker.Trigger(payload)
		return
	}
	// No workers available, waiting a while and retrying
	// TODO: some debug info?
	if retries >= l.maxRetries {
		<-time.After(l.retryDelay)
		retries += 1
		goto start
	}
}

// AddWorker pushes given worker to the list of the available workers. Threadsafe,
// may be called from many handlers and affects the other workers.
//
// worker - The worker to be added.
//
func (l *backendLobby) addWorker(worker *BackendWorker) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.workers[worker.Id()] = worker
	r := ring.New(1)
	r.Value = worker
	if l.robin == nil {
		l.robin = r
	} else {
		l.robin.Link(r)
	}
}

// deleteWorker removes specified worker from the load ballancer's ring.
// Threadsafe, may be called from many handlers and affects the other workers
// related calls.
//
// worker - The worker to be deleted
//
func (l *backendLobby) deleteWorker(worker *BackendWorker) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	worker.Kill()
	delete(l.workers, worker.Id())
}

// getWorkerById returns an worker with the specified ID and its existance
// status. Threadsafe, called only internally but may be affected by the other
// workers related calls.
func (l *backendLobby) getWorkerById(id string) (worker *BackendWorker, ok bool) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	worker, ok = l.workers[id]
	return
}

// getAvailableWorker pick an available agend and moves cursor forward (simple
// round robin technique).
//
// Returns an available worker.
func (l *backendLobby) getAvailableWorker() (worker *BackendWorker) {
	var ok bool
start:
	if l.robin == nil || l.robin.Len() == 0 {
		return
	}
	next := l.robin.Next()
	if worker, ok = next.Value.(*BackendWorker); !ok {
		return
	}
	if _, ok = l.getWorkerById(worker.Id()); !ok {
		// Seems that worker has been deleted, removing it from the load
		// ballancer's ring as well.
		l.robin.Unlink(1)
		worker = nil
		goto start
	} else {
		l.robin = next
	}
	return
}

// Exported
// -----------------------------------------------------------------------------

// Enqueue pushes given message to the queue.
//
// payload - data to be send to the client.
//
func (l *backendLobby) Enqueue(payload interface{}) {
	l.queue <- payload
}

// IsAlive returns whether this lobby is running or not.
func (l *backendLobby) IsAlive() bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	return l.queue != nil
}

// Kill stops execution of this lobby.
func (l *backendLobby) Kill() {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if l.queue != nil {
		close(l.queue)
		l.queue = nil
	}
}

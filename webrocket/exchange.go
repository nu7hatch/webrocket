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
	"log"
	"container/ring"
)

type exchange struct {
	workersQueue chan map[string]interface{}
	clientsQueue chan map[string]interface{}
	Log          *log.Logger
	workers      map[string]*mqConn
	clients      map[string]*wsConn
	robin       *ring.Ring
}

func newExchange(v *Vhost) *exchange {
	ex := new(exchange)
	ex.workersQueue = make(chan map[string]interface{})
	ex.clientsQueue = make(chan map[string]interface{})
	ex.workers = make(map[string]*mqConn)
	ex.clients = make(map[string]*wsConn)
	ex.Log = v.Log
	ex.robin = nil
	go ex.workersQueueLoop()
	go ex.clientsQueueLoop()
	v.exchange = ex
	return ex
}

func (ex *exchange) addWorker(id string, mq *mqConn) {
	ex.workers[id] = mq
	robin := ring.New(1)
	robin.Value = mq
	if ex.robin == nil {
		ex.robin = robin
	} else {
		ex.robin.Link(robin)
	}
}

func (ex *exchange) workersQueueLoop() {
	for {
		payload := <-ex.workersQueue
		if ex.robin == nil || ex.robin.Len() == 0 {
			println("no workers!")
			continue
		}
		conn, ok := ex.robin.Value.(*mqConn)
		if !ok {
			println(22)
			continue
		}
		conn.send(payload)
		println("sent to ", string(conn.id))
		ex.robin = ex.robin.Next()
	}
}

func (ex *exchange) clientsQueueLoop() {
	for {
		_ = <-ex.clientsQueue
		// ...
	}
}
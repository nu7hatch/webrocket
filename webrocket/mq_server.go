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
//	"fmt"
	"log"
//	zmq "../gozmq"
)

// The MQ exchange server manages backend app connections.
type MqServer struct {
	Addr   string
	Log    *log.Logger
	ctx    *Context
}

// Creates new MQ exchange bound to specified addr.
// A Trivial usage example
//
//     package main
//
//     import "webrocket"
//
//     func main() {
//         ctx := webrocket.NewContext()
//         srv := ctx.NewWebsocketServer(":9772")
//         mq := ctx.NewMqServer("localhost:9773")
//         ctx.AddVhost("/echo")
//         go mq.ListenAndServe()
//         srv.ListenAndServe()
//     }
//
func (ctx *Context) NewMqServer(addr string) *MqServer {
	mq := &MqServer{Addr: addr, ctx: ctx}
	mq.Log = ctx.Log
	ctx.mqServ = mq
	return mq
}

// Server's event loop.
func (mq *MqServer) eventLoop() {
	//for {
		//msg, _ := mq.socket.Recv(0)
		//fmt.Printf("%s", string(msg))
	//}
}

// ListenAndServe configures the ZeroMQ DEALER socket and starts
// server's event loop.
func (mq *MqServer) ListenAndServe() error {
	mq.Log.Printf("server[mq]: About to listen on %s\n", mq.Addr)
	return nil
	//mq.Log.Fatalf("server[mq]: Server startup error: %s\n", err.Error())
	//return err
}
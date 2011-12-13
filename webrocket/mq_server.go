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
	"encoding/json"
	zmq "../gozmq"
)

type mqConn struct {
	*conn
	id      []byte
	session *User
	vhost   *Vhost
	sock    *zmq.Socket
}

func newMqConn(sock *zmq.Socket, id []byte, user *User, vhost *Vhost) *mqConn {
	c := &mqConn{id: id}
	c.conn = new(conn)
	c.session = user
	c.vhost = vhost
	c.sock = sock
	vhost.exchange.addWorker(string(id), c)
	return c
}

func (mq *mqConn) send(payload interface{}) error {
	eventPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	(*mq.sock).Send(mq.id, zmq.SNDMORE)
	(*mq.sock).Send([]byte(eventPayload), 0)
	return nil
}

// The MQ exchange server manages backend app connections.
type MqServer struct {
	Addr     string
	Log      *log.Logger
	ctx      *Context
	router   zmq.Socket
	sessions map[string]*mqConn
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
	mq.sessions = make(map[string]*mqConn)
	ctx.mqServ = mq
	return mq
}

func (mq *MqServer) setupRouter(ctx *zmq.Context) error {
	var err error
	mq.router, err = (*ctx).NewSocket(zmq.ROUTER)
	if err == nil {
		err = mq.router.Bind(mq.Addr)
	}
	return err
}

func (mq *MqServer) dispatch(id []byte, payload []byte) {
	var data map[string]interface{}
	err := json.Unmarshal(payload, &data)
	if err != nil {
		mq.Log.Printf("server[mq]: ERR_INVALID_DATA_RECEIVED")
		return
	}
	msg, err := NewMessage(data)
	if err != nil {
		mq.Log.Printf("server[mq]: ERR_INVALID_MESSAGE_FORMAT")
		return
	}
	rdata, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}
	switch msg.Event {
	case "auth":
		vhostName, ok := rdata["vhost"].(string)
		if !ok {
			// invalid vhost name
			return
		}
		vhost, ok := mq.ctx.GetVhost(vhostName)
		if !ok {
			// invalid vhost
			return
		}
		userName, ok := rdata["user"].(string)
		if !ok || len(userName) == 0 {
			// invalid user name
			return
		}
		user, ok := vhost.GetUser(userName)
		if !ok {
			// user not found
			return
		}
		secret, ok := rdata["secret"].(string)
		if !ok {
			secret = ""
		}
		if !user.Authenticate(secret) {
			// unauthorized
			return
		}
		sendPayload, _ := json.Marshal(map[string]interface{}{"ok": true})
		mq.router.Send(id, zmq.SNDMORE)
		mq.router.Send([]byte(sendPayload), 0)
		conn := newMqConn(&mq.router, id, user, vhost)
		mq.sessions[string(id)] = conn
		println("authenticated as", user.Name, "on", vhost.path)
	case "broadcast":
		conn, ok := mq.sessions[string(id)]
		if !ok {
			// forbidden
			println(1)
			return
		}
		event, ok := rdata["event"].(string)
		if !ok || len(event) == 0 {
			// invalid event
			println(2)
			return
		}
		chanName, ok := rdata["channel"].(string)
		if !ok || len(chanName) == 0 {
			// invalid channel
			println(3)
			return 
		}
		sdata, ok := rdata["data"]
		if !ok {
			sdata = nil
		}
		channel := conn.vhost.GetOrCreateChannel(chanName)
		composedMsg := map[string]interface{}{event: sdata}
		channel.broadcast <- composedMsg
		println("Broadcasted!")
	}
}

func (mq *MqServer) Enqueue(payload map[string]interface{}) error {
	// ...
	return nil
}

func (mq *MqServer) ListenAndServe() error {
	var err error
	ctx, err := zmq.NewContext()
	if err == nil {
		mq.router, err = ctx.NewSocket(zmq.ROUTER)
		err = mq.router.Bind(mq.Addr)
		if err == nil {
			mq.Log.Printf("server[mq]: About to listen on %s", mq.Addr)
			for {
				id, _ := mq.router.Recv(zmq.SNDMORE)
				message, _ := mq.router.Recv(0)
				println(string(message))
				mq.dispatch(id, message)
			}
			return nil
		}
	}
	mq.Log.Fatalf("server[mq]: Server startup error: %s\n", err.Error())
	return err
}
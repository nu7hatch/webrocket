package webrocket

import (
	"net"
	"sync"
	"bufio"
)

type backendConnection struct {
	conn     net.Conn
	endpoint *BackendEndpoint
	mtx      sync.Mutex
}

func newBackendConnection(endpoint *BackendEndpoint, conn net.Conn) (c *backendConnection) {
	c = &backendConnection{
		conn:     conn,
		endpoint: endpoint,
	}
	return
}

func (c *backendConnection) Recv() (req *backendRequest, err error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	var msg = [][]byte{}
	var buf = bufio.NewReader(c.conn)
	var possibleEof = false
	for {
		chunk, err := buf.ReadSlice('\n')
		if err != nil {
			return nil, err
		}
		if string(chunk) == "\r\n" {
			if possibleEof {
				break
			}
			possibleEof = true
			continue
		} else {
			possibleEof = false
		}
		msg = append(msg[:], chunk[:len(chunk)-1])
	}
	if len(msg) < 3 {
		return
	}
	aid, cmd := msg[0], msg[2]
	req = newBackendRequest(c, nil, aid, string(cmd), msg[3:])
	return
}

func (c *backendConnection) Send(cmd string, frames ...string) (err error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	payload := cmd + "\n"
	for _, frame := range frames {
		payload += frame + "\n"
	}
	payload += "\r\n\r\n"
	_, err = c.conn.Write([]byte(payload))
	return
}

func (c *backendConnection) SetTimeout(nsec int64) {
	c.conn.SetReadTimeout(nsec)
}

func (c *backendConnection) Kill() {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.conn.Close()
}
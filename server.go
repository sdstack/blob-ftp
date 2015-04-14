package main

import (
	"net"
	"net/textproto"
)

type Conn struct {
	ctrl *textproto.Conn
	data net.Conn
	ln   net.Listener
}

func (c *Conn) Close() error {
	return c.ctrl.Close()
}

func NewServer(c net.Conn) (*Conn, error) {
	return &Conn{ctrl: textproto.NewConn(c)}, nil
}

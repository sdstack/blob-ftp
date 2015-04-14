package main

import (
	"net"
	"net/textproto"

	"github.com/ncw/swift"
)

type Conn struct {
	ctrl      *textproto.Conn
	data      net.Conn
	ln        net.Listener
	port      int
	mode      string
	path      string
	sw        *swift.Connection
	user      string
	token     string
	container string
	api       string
}

func (c *Conn) Close() error {
	return c.ctrl.Close()
}

func NewServer(c net.Conn) (*Conn, error) {
	return &Conn{api: "https://api.clodo.ru", user: "storage_21_1", token: "56652e9028ded5ea5d4772ba80e578ce", ctrl: textproto.NewConn(c)}, nil
}

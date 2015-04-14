package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func main() {
	l, err := net.Listen("tcp", ":8021")
	if err != nil {
		panic(err)

	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf(err.Error())
			continue
		}
		go handle(conn)
	}
}

func handle(c net.Conn) {
	defer c.Close()
	s, err := NewServer(c)
	if err != nil {
		fmt.Printf(err.Error())
	}
	s.ctrl.PrintfLine("220 %s [%s]", "go-ftp", c.LocalAddr())
	for {
		line, err := s.ctrl.ReadLine()
		if err != nil {
			fmt.Printf(err.Error())
			return
		}
		parts := strings.Fields(line)
		cmd := parts[0]
		params := parts[1:]
		switch cmd {
		case "USER":
			s.cmdServerUser(params)
		case "PASS":
			s.cmdServerPass(params)
		case "PWD":
			s.cmdServerPwd(params)
		case "EPSV":
			s.cmdServerEpsv(params)
		case "PASV":
			s.cmdServerPasv(params)
		case "TYPE":
			s.cmdServerType(params)
		case "LIST":
			s.cmdServerList(params)
		case "SYST":
			s.cmdServerSyst(params)
		case "QUIT":
			s.cmdServerQuit(params)
			s.Close()
			return
		case "SIZE":
			s.cmdServerSize(params)
		case "CWD":
			s.cmdServerCwd(params)
		case "RETR":
			s.cmdServerRetr(params)
		default:
			fmt.Printf("%s\n", line)
		}
	}
}

func (s *Conn) cmdServerUser(args []string) {
	fmt.Printf("cmdServerUser: %s\n", args)
	s.ctrl.PrintfLine("331 Password required for %s", args[0])
}

func (s *Conn) cmdServerPass(args []string) {
	fmt.Printf("cmdServerPass: %s\n", args)
	s.ctrl.PrintfLine("230 Logged on")
	//530 Login incorrect.
}

func (s *Conn) cmdServerPwd(args []string) {
	fmt.Printf("cmdServerPwd: %s\n", args)
	s.ctrl.PrintfLine(`257 "/" is current directory.`)
}

func (s *Conn) cmdServerEpsv(args []string) {
	fmt.Printf("cmdServerEpsv: %s\n", args)
	ln, err := net.Listen("tcp4", "37.139.40.30:0")
	if err != nil {

	}
	s.ln = ln
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	s.ctrl.PrintfLine("229 Entering Extended Passive Mode (|||%s|)", port)
	go func() {
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				return
			}
			s.data = conn
		}
	}()
}

func (s *Conn) cmdServerType(args []string) {
	fmt.Printf("cmdServerType: %s\n", args)
	switch args[0] {
	default:
		s.ctrl.PrintfLine("200 Type set to %s", args[0])
	}
}

func (s *Conn) cmdServerList(args []string) {
	fmt.Printf("cmdServerList: %s\n", args)
	s.ctrl.PrintfLine(`150 Opening data channel for directory listing of "%s"`, args[0])
	n, err := s.data.Write([]byte("-rw-r--r-- 1 ftp ftp              4 Apr 16  2008 TEST\n"))
	if err != nil {
		fmt.Printf(err.Error())
	}
	s.data.Close()
	s.ctrl.PrintfLine(`226 Closing data connection, sent %d bytes`, n)
	s.ctrl.PrintfLine("226 Transfer complete.")
}

func (s *Conn) cmdServerSyst(args []string) {
	fmt.Printf("cmdServerSyst: %s\n", args)
	s.ctrl.PrintfLine("215 UNIX Type: L8")
}

func (s *Conn) cmdServerQuit(args []string) {
	if s.data != nil {
		s.data.Close()
	}
	if s.ln != nil {
		s.ln.Close()
	}
	if s.ctrl != nil {
		s.ctrl.Close()
	}
}

func (s *Conn) cmdServerPasv(args []string) {
	fmt.Printf("cmdServerPasv: %s\n", args)

	ln, err := net.Listen("tcp4", "37.139.40.30:0")
	if err != nil {

	}
	s.ln = ln
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	p0, _ := strconv.Atoi(port)
	p1 := p0 / 256
	p2 := p0 - p1*256
	s.ctrl.PrintfLine("227 Entering Passive Mode (%s,%d,%d)", strings.Replace(host, ".", ",", -1), p1, p2)
	go func() {
		for {
			conn, err := s.ln.Accept()
			if err != nil {
				return
			}
			s.data = conn
		}
	}()
}

func (s *Conn) cmdServerSize(args []string) {
	fmt.Printf("cmdServerSize: %s\n", args)
	s.ctrl.PrintfLine("213 %d", len([]byte("test\n")))
}

func (s *Conn) cmdServerCwd(args []string) {
	fmt.Printf("cmdServerCwd: %s\n", args)
	s.ctrl.PrintfLine("250 Okay")
}

func (s *Conn) cmdServerRetr(args []string) {
	fmt.Printf("cmdServerRetr: %s\n", args)
	s.ctrl.PrintfLine(`150 Opening data channel for file contents "%s"`, args[0])
	n, err := s.data.Write([]byte("test\n"))
	if err != nil {
		fmt.Printf(err.Error())
	}
	s.data.Close()
	s.ctrl.PrintfLine(`226 Closing data connection, sent %d bytes`, n)
	s.ctrl.PrintfLine("226 Transfer complete.")

}

package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ncw/swift"
)

func main() {
	l, err := net.Listen("tcp4", ":21")
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
	//56652e9028ded5ea5d4772ba80e578ce
	//storage_21_1
	//https://api.clodo.ru/
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
		case "CWD", "XCWD":
			s.cmdServerCwd(params)
		case "RETR":
			s.cmdServerRetr(params)
		case "PORT":
			s.cmdServerPort(params)
		case "EPRT":
			s.cmdServerEprt(params)
		case "LPRT":
			s.cmdServerLprt(params)
		case "LPSV":
			s.cmdServerLpsv(params)
		case "FEAT":
			s.cmdServerFeat(params)
		default:
			fmt.Printf("%s\n", line)
		}
	}
}

func (s *Conn) cmdServerUser(args []string) {
	fmt.Printf("cmdServerUser: %s\n", args)
	//	s.user = args[0]
	s.ctrl.PrintfLine("331 Password required for %s", args[0])
}

func (s *Conn) cmdServerPass(args []string) {
	fmt.Printf("cmdServerPass: %s\n", args)
	//	s.token = args[0]
	s.sw = &swift.Connection{UserName: s.user, ApiKey: s.token, AuthUrl: s.api, AuthVersion: 1}
	err := s.sw.Authenticate()
	if err != nil {
		fmt.Printf(err.Error())
		s.ctrl.PrintfLine("530 Login incorrect")
		return
	}

	s.ctrl.PrintfLine("230 Logged on")
}

func (s *Conn) cmdServerPwd(args []string) {
	fmt.Printf("cmdServerPwd: %s\n", args)
	if s.path == "" {
		s.path = "/"
	}
	s.ctrl.PrintfLine(`257 "%s" is current directory.`, s.path)
}

func (s *Conn) cmdServerEpsv(args []string) {
	fmt.Printf("cmdServerEpsv: %s\n", args)
	if s.ln == nil {
		ln, err := net.Listen("tcp4", fmt.Sprintf(":%d", s.port))
		if err != nil {
			fmt.Printf(err.Error())
			s.ctrl.PrintfLine("425 Data connection failed")
		}
		s.ln = ln
	}
	if s.port == 0 {
		_, port, _ := net.SplitHostPort(s.ln.Addr().String())
		s.port, _ = strconv.Atoi(port)
	}
	s.ctrl.PrintfLine("229 Entering Extended Passive Mode (|||%d|)", s.port)
}

func (s *Conn) cmdServerType(args []string) {
	fmt.Printf("cmdServerType: %s\n", args)
	s.mode = args[0]
	s.ctrl.PrintfLine("200 Type set to %s", s.mode)
}

func (s *Conn) cmdServerList(args []string) {
	fmt.Printf("cmdServerList: %s\n", args)
	s.ctrl.PrintfLine(`150 Opening data channel for directory listing of "%s"`, strings.Join(args, ""))
	c, err := s.newPassiveSocket()
	if err != nil {
		fmt.Printf(err.Error())
		s.ctrl.PrintfLine("425 Data connection failed")
		return
	}
	var n int
	var files []os.FileInfo

	if s.path == "/" {
		cnts, err := s.sw.ContainersAll(nil)
		if err != nil {
			fmt.Printf(err.Error())
			return
		}
		files = append(files, NewDirItem(".", 4096, 0), NewDirItem("..", 4096, 0))
		for _, cnt := range cnts {
			it := NewDirItem(cnt.Name, cnt.Bytes, cnt.Count)
			files = append(files, it)
		}
	} else {
		if idx := strings.Index(s.path, "/"); idx != -1 {
			s.container = s.path[idx+1:]
		} else {
			s.container = s.path
		}
		objs, err := s.sw.ObjectsAll(s.container, nil)
		if err != nil {
			fmt.Printf(err.Error())
			return
		}
		files = append(files, NewDirItem(".", 4096, 0), NewDirItem("..", 4096, 0))
		for _, obj := range objs {
			it := NewFileItem(obj.Name, obj.Bytes, 1)
			files = append(files, it)
		}
	}

	ls := newListFormatter(files)
	n, err = c.Write([]byte(ls.Detailed()))
	if err != nil {
		fmt.Printf(err.Error())
	}
	c.Close()
	s.ctrl.PrintfLine(`226 Closing data connection, sent %d bytes`, n)
	//	s.ctrl.PrintfLine("226 Transfer complete.")
}

func (s *Conn) cmdServerSyst(args []string) {
	fmt.Printf("cmdServerSyst: %s\n", args)
	s.ctrl.PrintfLine("215 UNIX Type: L8")
}

func (s *Conn) cmdServerQuit(args []string) {
	s.ctrl.PrintfLine("221 Goodbye")
	if s.ln != nil {
		s.ln.Close()
	}
	if s.ctrl != nil {
		s.ctrl.Close()
	}
}

func (s *Conn) cmdServerPasv(args []string) {
	fmt.Printf("cmdServerPasv: %s\n", args)
	if s.ln == nil {
		ln, err := net.Listen("tcp4", fmt.Sprintf(":%d", s.port))
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			s.ctrl.PrintfLine("425 Data connection failed")
		}
		s.ln = ln
	}
	host, port, _ := net.SplitHostPort(s.ln.Addr().String())
	p0, _ := strconv.Atoi(port)
	p1 := p0 / 256
	p2 := p0 - p1*256
	s.port, _ = strconv.Atoi(port)
	s.ctrl.PrintfLine("227 Entering Passive Mode (%s,%d,%d)", strings.Replace(host, ".", ",", -1), p1, p2)
}

func (s *Conn) newPassiveSocket() (net.Conn, error) {
	return s.ln.Accept()
}

func (s *Conn) cmdServerSize(args []string) {
	fmt.Printf("cmdServerSize: %s\n", args)
	s.ctrl.PrintfLine("213 %d", len([]byte("test\n")))
}

func (s *Conn) cmdServerCwd(args []string) {
	fmt.Printf("cmdServerCwd: %s\n", args)
	if len(args) == 0 {
		s.path = "/"
	} else {
		s.path = filepath.Clean(args[0])
	}
	s.ctrl.PrintfLine(`250 "%s" is current directory.`, s.path)

}

func (s *Conn) cmdServerPort(args []string) {
	fmt.Printf("cmdServerPort: %s\n", args)
	s.ctrl.PrintfLine("501 Data connection failed")
}

func (s *Conn) cmdServerEprt(args []string) {
	fmt.Printf("cmdServerEprt: %s\n", args)
	s.ctrl.PrintfLine("501 Data connection failed")
}

func (s *Conn) cmdServerLprt(args []string) {
	fmt.Printf("cmdServerLprt: %s\n", args)
	s.ctrl.PrintfLine("501 Data connection failed")
}

func (s *Conn) cmdServerLpsv(args []string) {
	fmt.Printf("cmdServerLpsv: %s\n", args)
	s.ctrl.PrintfLine("501 Data connection failed")
}

func (s *Conn) cmdServerFeat(args []string) {
	fmt.Printf("cmdServerFeat: %s\n", args)
	s.ctrl.PrintfLine("501 Data connection failed")
	return
	s.ctrl.PrintfLine(`211-Extensions supported:
    SIZE
    RETR
    CWD
    PWD
    QUIT
    SYST
    PASV
    EPSV
    LIST
    TYPE
    USER
    PASS
    UTF8
211 END`)
}

func (s *Conn) cmdServerRetr(args []string) {
	fmt.Printf("cmdServerRetr: %s\n", args)
	s.ctrl.PrintfLine(`150 Opening data channel for file contents "%s"`, args[0])
	c, err := s.newPassiveSocket()
	if err != nil {
		s.ctrl.PrintfLine("425 Data connection failed")
		return
	}
	n, err := c.Write([]byte("test\n"))
	if err != nil {
		s.ctrl.PrintfLine("425 Data connection failed")
		fmt.Printf(err.Error())
		return
	}
	c.Close()
	s.ctrl.PrintfLine(`226 Closing data connection, sent %d bytes`, n)
	s.ctrl.PrintfLine("226 Transfer complete.")

}

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
		params := strings.Join(parts[1:], "")
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
		case "DELE":
			s.cmdServerDele(params)
		case "MKD", "XMKD":
			s.cmdServerMkd(params)
		case "RMD", "XRMD":
			s.cmdServerRmd(params)
		case "STOR":
			s.cmdServerStor(params)
		case "SITE":
			s.cmdServerSite(params)
		default:
			fmt.Printf("%s\n", line)
		}
	}
}

func (s *Conn) cmdServerUser(args string) {
	fmt.Printf("cmdServerUser: %s\n", args)
	//	s.user = args[0]
	s.ctrl.PrintfLine("331 Password required for %s", args)
}

func (s *Conn) cmdServerPass(args string) {
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

func (s *Conn) cmdServerPwd(args string) {
	fmt.Printf("cmdServerPwd: %s\n", args)
	s.ctrl.PrintfLine(`257 "%s" is current directory.`, s.path)
}

func (s *Conn) cmdServerEpsv(args string) {
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
	s.passive = true
	s.ctrl.PrintfLine("229 Entering Extended Passive Mode (|||%d|)", s.port)
}

func (s *Conn) cmdServerType(args string) {
	fmt.Printf("cmdServerType: %s\n", args)
	s.mode = args
	s.ctrl.PrintfLine("200 Type set to %s", s.mode)
}

func (s *Conn) cmdServerList(args string) {
	fmt.Printf("cmdServerList: %s\n", args)
	s.ctrl.PrintfLine(`150 Opening data channel for directory listing of "%s"`, args)
	c, err := s.newSocket()
	if err != nil {
		fmt.Printf(err.Error())
		s.ctrl.PrintfLine("425 Data connection failed")
		return
	}
	var files []os.FileInfo
	cnt := ""
	p := ""
	switch args {
	case "-la", "-l", "-a", "-al":
		args = ""
	}
	if len(args) > 0 && args[0] == '/' {
		cnt = strings.Split(args, "/")[1]
		p = strings.Join(strings.Split(args, "/")[2:], "/")
	}

	if cnt == "" && s.path == "/" {
		cnts, err := s.sw.ContainersAll(nil)
		if err != nil {
			fmt.Printf(err.Error())
			return
		}
		files = append(files, NewDirItem(".", 4096, 0), NewDirItem("..", 4096, 0))
		for _, ct := range cnts {
			it := NewDirItem(ct.Name, ct.Bytes, ct.Count)
			files = append(files, it)
		}
	} else {
		opts := &swift.ObjectsOpts{Delimiter: '/'}
		cnt = strings.Split(s.path, "/")[1]
		p = filepath.Clean(filepath.Join(strings.Join(strings.Split(s.path, "/")[2:], "/"), args))

		if p != "." {
			opts.Path = p
		}
		objs, err := s.sw.ObjectsAll(cnt, opts)
		if err != nil {
			fmt.Printf(err.Error())
			return
		}

		files = append(files, NewDirItem(".", 4096, 0), NewDirItem("..", 4096, 0))
		var it os.FileInfo

		for _, obj := range objs {
			if obj.PseudoDirectory || obj.ContentType == "application/directory" {
				it = NewDirItem(obj.Name, obj.Bytes, 1)
			} else {
				it = NewFileItem(obj.Name, obj.Bytes, 1)
			}
			files = append(files, it)
		}
	}
	ls := newListFormatter(files)
	_, err = c.Write([]byte(ls.Detailed()))
	if err != nil {
		fmt.Printf(err.Error())
	}
	c.Close()
	s.ctrl.PrintfLine(`226 Closing data connection`)
}

func (s *Conn) cmdServerSyst(args string) {
	fmt.Printf("cmdServerSyst: %s\n", args)
	s.ctrl.PrintfLine("215 UNIX Type: L8")
}

func (s *Conn) cmdServerQuit(args string) {
	s.ctrl.PrintfLine("221 Goodbye")
	if s.ln != nil {
		s.ln.Close()
	}
	if s.ctrl != nil {
		s.ctrl.Close()
	}
}

func (s *Conn) cmdServerPasv(args string) {
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
	s.passive = true
	s.ctrl.PrintfLine("227 Entering Passive Mode (%s,%d,%d)", strings.Replace(host, ".", ",", -1), p1, p2)
}

func (s *Conn) newPassiveSocket() (net.Conn, error) {
	return s.ln.Accept()
}

func (s *Conn) newActiveSocket() (net.Conn, error) {
	return net.Dial("tcp", fmt.Sprintf("%s:%d", s.host, s.port))
}

func (s *Conn) newSocket() (net.Conn, error) {
	if s.passive {
		return s.newPassiveSocket()
	}
	return s.newActiveSocket()
}

func (s *Conn) cmdServerSize(args string) {
	fmt.Printf("cmdServerSize: %s\n", args)

	if s.path == "/" && len(args) == 0 {
		s.ctrl.PrintfLine("213 %d", 0)
		return
	}

	p := ""
	cnt := ""

	if args[0] == '/' {
		cnt = strings.Split(args, "/")[1]
		p = strings.Join(strings.Split(args, "/")[2:], "")
	} else {
		if s.path != "/" {
			cnt = strings.Split(s.path, "/")[1]
			p = filepath.Clean(filepath.Join(strings.Join(strings.Split(s.path, "/")[2:], "/"), args))
		} else {
			cnt = strings.Split(args, "/")[0]
		}
	}
	if p != "" {
		obj, _, err := s.sw.Object(cnt, p)
		if err != nil {
			fmt.Printf(err.Error())
			s.ctrl.PrintfLine("501 Error")
			return
		}
		s.ctrl.PrintfLine("213 %d", obj.Bytes)
	} else {
		s.ctrl.PrintfLine("550 Could not get file size.")
	}
}

func (s *Conn) cmdServerCwd(args string) {
	fmt.Printf("cmdServerCwd: %s\n", args)
	s.path = filepath.Clean("/" + args)
	s.ctrl.PrintfLine(`250 "%s" is current directory.`, s.path)

}

func (s *Conn) cmdServerPort(args string) {
	fmt.Printf("cmdServerPort: %s\n", args)
	s.passive = false
	s.ctrl.PrintfLine("522 Data connection failed")
}

func (s *Conn) cmdServerEprt(args string) {
	fmt.Printf("cmdServerEprt: %s\n", args)
	parts := strings.Split(args, "|")
	s.host = parts[2]
	s.port, _ = strconv.Atoi(parts[3])

	s.passive = false
	s.ctrl.PrintfLine("200 Ok")
}

func (s *Conn) cmdServerLprt(args string) {
	fmt.Printf("cmdServerLprt: %s\n", args)
	s.ctrl.PrintfLine("522 Data connection failed")
}

func (s *Conn) cmdServerLpsv(args string) {
	fmt.Printf("cmdServerLpsv: %s\n", args)
	s.ctrl.PrintfLine("522 Data connection failed")
}

func (s *Conn) cmdServerFeat(args string) {
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

func (s *Conn) cmdServerDele(args string) {
	fmt.Printf("cmdServerDele: %s\n", args)
	p := ""
	cnt := ""
	var err error

	if args[0] == '/' {
		cnt = strings.Split(args, "/")[1]
		p = filepath.Clean(strings.Join(strings.Split(args, "/")[2:], "/"))
	} else {
		if s.path != "/" {
			cnt = strings.Split(s.path, "/")[1]
			p = filepath.Clean(filepath.Join(strings.Join(strings.Split(s.path, "/")[2:], "/"), args))
		}
	}
	err = s.sw.ObjectDelete(cnt, p)
	if err != nil {
		fmt.Printf(err.Error())
		s.ctrl.PrintfLine(`450 Delete failed "%s"`, args)
		return
	}
	s.ctrl.PrintfLine(`250 Delete success "%s"`, args)
}

func (s *Conn) cmdServerRmd(args string) {
	fmt.Printf("cmdServerRmd: %s\n", args)
	p := ""
	cnt := ""
	var err error

	if args[0] == '/' {
		cnt = strings.Split(args, "/")[1]
		p = filepath.Clean(strings.Join(strings.Split(args, "/")[2:], "/"))
	} else {
		if s.path != "/" {
			cnt = strings.Split(s.path, "/")[1]
			p = filepath.Clean(filepath.Join(strings.Join(strings.Split(s.path, "/")[2:], "/"), args))
		}
	}
	if p != "" && p != "." {
		err = s.sw.ObjectDelete(cnt, p)
	} else {
		err = s.sw.ContainerDelete(cnt)
	}
	if err != nil {
		fmt.Printf(err.Error())
		s.ctrl.PrintfLine(`450 Delete failed "%s"`, args)
		return
	}
	s.ctrl.PrintfLine(`250 Delete success "%s"`, args)
}

func (s *Conn) cmdServerMkd(args string) {
	fmt.Printf("cmdServerMkd: %s\n", args)
	p := ""
	cnt := ""
	var err error
	if args[0] == '/' {
		cnt = strings.Split(args, "/")[1]
		p = filepath.Clean(strings.Join(strings.Split(args, "/")[2:], "/"))
	} else {
		if s.path != "/" {
			cnt = strings.Split(s.path, "/")[1]
			p = filepath.Clean(filepath.Join(strings.Join(strings.Split(s.path, "/")[2:], "/"), args))
		}
	}

	if p != "" && p != "." {
		var f *swift.ObjectCreateFile
		f, err = s.sw.ObjectCreate(cnt, p, false, "", "application/directory", nil)
		if err != nil {
			fmt.Printf(err.Error())
			s.ctrl.PrintfLine(`550 Create failed "%s"`, args)
			return
		}
		err = f.Close()
	} else {
		err = s.sw.ContainerCreate(cnt, nil)
	}
	if err != nil {
		fmt.Printf(err.Error())
		s.ctrl.PrintfLine(`550 Create failed "%s"`, args)
		return
	}
	s.ctrl.PrintfLine(`250 Create success "%s"`, args)
}

func (s *Conn) cmdServerRetr(args string) {
	fmt.Printf("cmdServerRetr: %s\n", args)

	p := ""
	cnt := ""

	if args[0] == '/' {
		cnt = strings.Split(args, "/")[1]
		p = filepath.Clean(strings.Join(strings.Split(args, "/")[2:], "/"))
	} else {
		cnt = strings.Split(s.path, "/")[1]
		p = filepath.Clean(filepath.Join(strings.Join(strings.Split(s.path, "/")[2:], "/"), args))
	}

	_, _, err := s.sw.Object(cnt, p)
	if err != nil {
		s.ctrl.PrintfLine("550 Failed to open file")
		return
	}

	s.ctrl.PrintfLine(`150 Opening data channel for file contents "%s"`, args)
	c, err := s.newSocket()
	if err != nil {
		s.ctrl.PrintfLine("425 Data connection failed")
		return
	}
	defer c.Close()

	_, err = s.sw.ObjectGet(cnt, p, c, false, nil)
	if err != nil {
		s.ctrl.PrintfLine("550 Failed to open file")
		return
	}

	s.ctrl.PrintfLine(`226 Closing data connection`)
}

func (s *Conn) cmdServerStor(args string) {
	fmt.Printf("cmdServerRetr: %s\n", args)

	p := ""
	cnt := ""

	if args[0] == '/' {
		cnt = strings.Split(args, "/")[1]
		p = filepath.Clean(strings.Join(strings.Split(args, "/")[2:], "/"))
	} else {
		cnt = strings.Split(s.path, "/")[1]
		p = filepath.Clean(filepath.Join(strings.Join(strings.Split(s.path, "/")[2:], "/"), args))
	}

	s.ctrl.PrintfLine(`150 Opening data channel for file contents "%s"`, args)
	c, err := s.newSocket()
	if err != nil {
		s.ctrl.PrintfLine("425 Data connection failed")
		return
	}
	defer c.Close()

	_, err = s.sw.ObjectPut(cnt, p, c, false, "", "", nil)
	if err != nil {
		s.ctrl.PrintfLine("552 Failed to store file")
		return
	}

	s.ctrl.PrintfLine(`226 Closing data connection`)
}

func (s *Conn) cmdServerSite(args string) {
	s.ctrl.PrintfLine(`200 Success`)
	//SITE CHMOD 0644 /public/.wgetpaste.conf
}

package main

import (
	"database/sql"
	"fmt"
	"net"
	"net/textproto"

	_ "github.com/cznic/ql/driver"
	"github.com/ncw/swift"
)

var schema []string = []string{
	`CREATE TABLE IF NOT EXISTS containers (ID int, Name string, Bytes int, Count int)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS containersID ON containers (ID)`,
	`CREATE INDEX IF NOT EXISTS containersName ON containers (Name)`,
	`CREATE TABLE IF NOT EXISTS objects (ID int, Container string, Prefix string, ContentType string, LastModified time, Name string, Bytes int, Count int, SubDir string, PseudoDir bool)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS objectsID ON objects (ID)`,
	`CREATE INDEX IF NOT EXISTS objectsName ON objects (Name)`,
	`CREATE INDEX IF NOT EXISTS objectsContainer ON objects (Container)`,
}

type Conn struct {
	ctrl    *textproto.Conn
	data    net.Conn
	ln      net.Listener
	host    string
	port    int
	mode    string
	sw      *swift.Connection
	user    string
	token   string
	path    string
	api     string
	passive bool
	movesrc string
	movedst string
	db      *sql.DB
}

func (c *Conn) Close() error {
	c.db.Close()
	return c.ctrl.Close()
}

func NewServer(c net.Conn) (*Conn, error) {
	var err error
	conn := &Conn{api: "https://api.clodo.ru", user: "storage_21_1", token: "56652e9028ded5ea5d4772ba80e578ce", ctrl: textproto.NewConn(c), path: "/"}
	conn.db, err = sql.Open("ql", "memory://mem.db")
	if err != nil {
		return nil, err
	}
	tx, err := conn.db.Begin()
	if err != nil {
		conn.db.Close()
		return nil, err
	}
	for _, query := range schema {
		if _, err := tx.Exec(query); err != nil {
			conn.db.Close()
			fmt.Printf("%s\n", err.Error())
			return nil, fmt.Errorf("q: %s e: %s", query, err.Error())
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *Conn) saveContainers(cnts []swift.Container) error {
	fmt.Printf("%+v\n", cnts)
	tx, err := c.db.Begin()
	if err != nil {
		fmt.Printf("777 %s\n", err.Error())
		return err
	}

	for _, cnt := range cnts {
		if _, err = tx.Exec(`INSERT INTO containers (Name, Bytes, Count) VALUES ($1, $2, $3)`, cnt.Name, cnt.Bytes, cnt.Count); err != nil {
			fmt.Printf("9999 %s\n", err.Error())
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		fmt.Printf("ssss %s\n", err.Error())
		return err
	}

	return nil
}

func (c *Conn) saveObjects(cnt string, objs []swift.Object, opts *swift.ObjectsOpts) error {
	fmt.Printf("%+v\n", objs)
	tx, err := c.db.Begin()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return err
	}

	for _, obj := range objs {
		if _, err = tx.Exec(`INSERT INTO objects (Container, Prefix, ContentType, LastModified,  Name, Bytes, Count, PseudoDir, SubDir) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, cnt, opts.Prefix, obj.ContentType, obj.LastModified, obj.Name, obj.Bytes, 1, obj.PseudoDirectory, obj.SubDir); err != nil {
			fmt.Printf("aaaa %s\n", err.Error())
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		fmt.Printf("%s\n", err.Error())
		return err
	}
	return nil
}

func (c *Conn) loadObjects(cnt string, opts *swift.ObjectsOpts) ([]swift.Object, error) {
	var objs []swift.Object
	var res swift.Object

	result, err := c.db.Query(`SELECT ContentType, LastModified, Name, Bytes, PseudoDir, SubDir from objects where Container==$1 && Prefix==$2`, cnt, opts.Prefix)
	if err != nil {
		fmt.Printf("bbb %s\n", err.Error())
		return objs, err
	}
	defer result.Close()

	for result.Next() {
		if err = result.Scan(&res.ContentType, &res.LastModified, &res.Name, &res.Bytes, &res.PseudoDirectory, &res.SubDir); err != nil {
			fmt.Printf("%s\n", err.Error())
			return objs, err
		}
		objs = append(objs, res)
	}

	if err := result.Err(); err != nil {
		fmt.Printf("%s\n", err.Error())
		return objs, err
	}
	if len(objs) < 1 {
		return objs, fmt.Errorf("empty")
	}
	return objs, nil
}

func (c *Conn) loadContainers() ([]swift.Container, error) {
	var cnts []swift.Container
	var res swift.Container

	result, err := c.db.Query(`SELECT Name, Bytes, Count from containers`)
	if err != nil {
		fmt.Printf("111 %s\n", err.Error())
		return cnts, err
	}
	defer result.Close()

	for result.Next() {
		if err = result.Scan(&res.Name, &res.Bytes, &res.Count); err != nil {
			fmt.Printf("222 %s\n", err.Error())
			return cnts, err
		}
		cnts = append(cnts, res)
	}

	if err := result.Err(); err != nil {
		fmt.Printf("333 %s\n", err.Error())
		return cnts, err
	}
	if len(cnts) < 1 {
		return cnts, fmt.Errorf("empty")
	}
	fmt.Printf("444 %+v\n", cnts)
	return cnts, nil
}

package main

import (
	"os"
	"time"
)

type FileInfo struct {
	name  string
	bytes int64
	links int64
	mode  os.FileMode
}

func (info *FileInfo) Name() string {
	return info.name
}

func (info *FileInfo) Size() int64 {
	return info.bytes
}

func (info *FileInfo) Mode() os.FileMode {
	return info.mode
}

func (info *FileInfo) ModTime() time.Time {
	return time.Now()
}

func (info *FileInfo) IsDir() bool {
	return (info.mode | os.ModeDir) == os.ModeDir
}

func (info *FileInfo) Sys() interface{} {
	return nil
}

func NewDirItem(name string, bytes int64, links int64) os.FileInfo {
	d := new(FileInfo)
	d.name = name
	d.bytes = int64(bytes)
	d.links = int64(links)
	d.mode = os.ModeDir | 0777
	return d
}

func NewFileItem(name string, bytes int64, links int64) os.FileInfo {
	f := new(FileInfo)
	f.name = name
	f.bytes = int64(bytes)
	f.links = int64(links)
	f.mode = 0666
	return f
}

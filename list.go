package main

import (
	"fmt"
	"os"

	"github.com/jehiah/go-strftime"
)

type listFormatter struct {
	files []os.FileInfo
}

func newListFormatter(files []os.FileInfo) *listFormatter {
	f := new(listFormatter)
	f.files = files
	return f
}

// Short returns a string that lists the collection of files by name only,
// one per line
func (formatter *listFormatter) Short() string {
	output := ""
	for _, file := range formatter.files {
		output += file.Name() + "\r\n"
	}
	output += "\r\n"
	return output
}

// Detailed returns a string that lists the collection of files with extra
// detail, one per line
func (formatter *listFormatter) Detailed() string {
	output := ""
	for _, file := range formatter.files {
		output += fmt.Sprintf("%-13s %s %-8s %-8s %8d %s %s\r\n", file.Mode().String(), "1", "1000", "1000", int(file.Size()), strftime.Format("%b %m  %Y", file.ModTime()), file.Name())
	}
	return output
}

package codegen

import (
	"strconv"
)

type Import struct {
	Name  string
	Alias string
	Path  string
}

type Imports struct {
	imports []*Import
	destDir string
}

func (i *Import) Write() string {
	return i.Alias + " " + strconv.Quote(i.Path)
}

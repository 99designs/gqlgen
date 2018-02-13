package codegen

import (
	"strconv"
)

type Import struct {
	Name    string
	Package string
}

type Imports []*Import

func (i *Import) Write() string {
	return i.Name + " " + strconv.Quote(i.Package)
}

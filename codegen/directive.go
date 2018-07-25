package codegen

import "strings"

type Directive struct {
	Name string
}

func (d *Directive) GoName() string {
	return strings.Title(d.Name)
}

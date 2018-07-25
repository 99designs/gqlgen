package codegen

import "strings"

type Directive struct {
	name string
}

func (d *Directive) Name() string {
	return strings.Title(d.name)
}

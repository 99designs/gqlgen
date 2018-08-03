package codegen

import (
	"fmt"
	"strconv"
	"strings"
)

type Directive struct {
	Name string
	Args []FieldArgument
}

func (d *Directive) CallArgs() string {
	args := []string{"ctx", "n"}

	for _, arg := range d.Args {
		args = append(args, "args["+strconv.Quote(arg.GQLName)+"].("+arg.Signature()+")")
	}

	return strings.Join(args, ", ")
}

func (d *Directive) Declaration() string {
	res := ucFirst(d.Name) + " func(ctx context.Context, next graphql.Resolver"

	for _, arg := range d.Args {
		res += fmt.Sprintf(", %s %s", arg.GoVarName, arg.Signature())
	}

	res += ") (res interface{}, err error)"
	return res
}

package unified

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
)

type Directive struct {
	Name string
	Args []FieldArgument
}

func (d *Directive) ArgsFunc() string {
	if len(d.Args) == 0 {
		return ""
	}

	return "dir_" + d.Name + "_args"
}

func (d *Directive) CallArgs() string {
	args := []string{"ctx", "obj", "n"}

	for _, arg := range d.Args {
		args = append(args, "args["+strconv.Quote(arg.GQLName)+"].("+templates.CurrentImports.LookupType(arg.GoType)+")")
	}

	return strings.Join(args, ", ")
}

func (d *Directive) ResolveArgs(obj string, next string) string {
	args := []string{"ctx", obj, next}

	for _, arg := range d.Args {
		dArg := "&" + arg.GoVarName
		if !arg.IsPtr() {
			if arg.Value != nil {
				dArg = templates.Dump(arg.Value)
			} else {
				dArg = templates.Dump(arg.Default)
			}
		} else if arg.Value == nil && arg.Default == nil {
			dArg = "nil"
		}

		args = append(args, dArg)
	}

	return strings.Join(args, ", ")
}

func (d *Directive) Declaration() string {
	res := ucFirst(d.Name) + " func(ctx context.Context, obj interface{}, next graphql.Resolver"

	for _, arg := range d.Args {
		res += fmt.Sprintf(", %s %s", arg.GoVarName, templates.CurrentImports.LookupType(arg.GoType))
	}

	res += ") (res interface{}, err error)"
	return res
}

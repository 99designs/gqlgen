package codegen

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Directives []*Directive

// NoImplementation returns the set of Directives without a preconfigured implementation
func (ds Directives) NoImplementation() Directives {
	noImpl := Directives{}
	for _, d := range ds {
		if d.Implementation == nil {
			noImpl = append(noImpl, d)
		}
	}
	return noImpl
}

type Directive struct {
	Name           string
	Args           []FieldArgument
	Implementation *Ref
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
		args = append(args, "args["+strconv.Quote(arg.GQLName)+"].("+arg.Signature()+")")
	}

	return strings.Join(args, ", ")
}

func (d *Directive) Declaration() string {
	return ucFirst(d.Name) + " " + d.Signature()
}

func (d *Directive) Signature() string {
	res := "func(ctx context.Context, obj interface{}, next graphql.Resolver"

	for _, arg := range d.Args {
		res += fmt.Sprintf(", %s %s", arg.GoVarName, arg.Signature())
	}

	res += ") (res interface{}, err error)"
	return res
}

func (d *Directive) validateSignature(sig *types.Signature) error {
	params := sig.Params()
	if params.Len() != len(d.Args)+3 {
		return errors.Errorf("param count mismatch (%d)", params.Len())
	}
	expected := []string{"context.Context", "interface{}", "github.com/99designs/gqlgen/graphql.Resolver"}
	for _, arg := range d.Args {
		expected = append(expected, arg.FullSignature())
	}
	for i, t := range expected {
		param := params.At(i)
		if param.Type().String() != t {
			return errors.Errorf("%s expected %s actual %s", param.Name(), t, param.Type().String())
		}
	}
	if sig.Results().Len() != 2 || sig.Results().At(0).Type().String() != "interface{}" || sig.Results().At(1).Type().String() != "error" {
		return errors.Errorf("invalid return types")
	}

	return nil
}

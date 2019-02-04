package codegen

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

type ArgSet struct {
	Args     []*FieldArgument
	FuncDecl string
}

func (b *builder) buildArg(obj *Object, arg *ast.ArgumentDefinition) (*FieldArgument, error) {
	argDirs, err := b.getDirectives(arg.Directives)
	if err != nil {
		return nil, err
	}
	newArg := FieldArgument{
		GQLName:       arg.Name,
		TypeReference: b.NamedTypes.getType(arg.Type),
		Object:        obj,
		GoVarName:     templates.ToGoPrivate(arg.Name),
		Directives:    argDirs,
	}

	if !newArg.TypeReference.Definition.GQLDefinition.IsInputType() {
		return nil, errors.Errorf(
			"cannot use %s as argument %s because %s is not a valid input type",
			newArg.Definition.GQLDefinition.Name,
			arg.Name,
			newArg.TypeReference.Definition.GQLDefinition.Kind,
		)
	}

	if arg.DefaultValue != nil {
		newArg.Default, err = arg.DefaultValue.Value(nil)
		if err != nil {
			return nil, errors.Errorf("default value is not valid: %s", err.Error())
		}
	}

	return &newArg, nil
}

func (b *builder) bindArgs(field *Field, params *types.Tuple) error {
	var newArgs []*FieldArgument

nextArg:
	for j := 0; j < params.Len(); j++ {
		param := params.At(j)
		for _, oldArg := range field.Args {
			if strings.EqualFold(oldArg.GQLName, param.Name()) {
				oldArg.TypeReference.GoType = param.Type()
				newArgs = append(newArgs, oldArg)
				continue nextArg
			}
		}

		// no matching arg found, abort
		return fmt.Errorf("arg %s not found on method", param.Name())
	}

	field.Args = newArgs
	return nil
}

func (a *Data) Args() map[string][]*FieldArgument {
	ret := map[string][]*FieldArgument{}
	for _, o := range a.Objects {
		for _, f := range o.Fields {
			if len(f.Args) > 0 {
				ret[f.ArgsFunc()] = f.Args
			}
		}
	}

	for _, d := range a.Directives {
		if len(d.Args) > 0 {
			ret[d.ArgsFunc()] = d.Args
		}
	}
	return ret
}

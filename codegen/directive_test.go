package codegen

import (
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
)

func TestDirectiveCallArgs(t *testing.T) {
	d := &Directive{
		Args: []*FieldArgument{
			{
				ArgumentDefinition: &ast.ArgumentDefinition{
					Name: "def1",
				},
				TypeReference: &config.TypeReference{
					GO: types.Default(
						types.NewNamed(
							types.NewTypeName(0, nil, "string", nil),
							types.Typ[types.String],
							nil,
						),
					),
				},
			},
			{
				ArgumentDefinition: &ast.ArgumentDefinition{
					Name: "def2",
				},
				TypeReference: &config.TypeReference{
					GO: types.Default(types.NewStruct(nil, nil)),
				},
			},
		},
	}

	got := d.CallArgs()

	assert.Equal(t, `ctx, obj, n, args["def1"].(string), args["def2"].(struct{})`, got)
}

func TestDirectiveParamNames(t *testing.T) {
	args := func(names ...string) []*FieldArgument {
		list := make([]*FieldArgument, len(names))
		for i, name := range names {
			list[i] = &FieldArgument{ArgumentDefinition: &ast.ArgumentDefinition{Name: name}}
		}
		return list
	}

	t.Run("the parameters and results before the arguments get a suffix", func(t *testing.T) {
		assert.Equal(t,
			[]string{"ctxArg", "objArg", "nextArg", "resArg", "errArg", "value"},
			directiveParamNames(args("ctx", "obj", "next", "res", "err", "value")))
	})

	t.Run("a parameter does not repeat an earlier one", func(t *testing.T) {
		assert.Equal(t,
			[]string{"fooBar", "fooBarArg", "objArg", "objArgArg", "_", "_"},
			directiveParamNames(args("foo_bar", "fooBar", "obj", "objArg", "_", "_")))
	})
}

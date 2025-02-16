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
					GO: types.Default(types.NewNamed(types.NewTypeName(0, nil, "string", nil), types.Typ[types.String], nil)),
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

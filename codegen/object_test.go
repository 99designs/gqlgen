package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestObjectInvalidsIncrement(t *testing.T) {
	// non-concurrent: all fields are non-resolver, non-method-with-context
	sequential := &Object{Definition: &ast.Definition{Name: "Query"}}
	sequential.Fields = []*Field{
		{FieldDefinition: &ast.FieldDefinition{Name: "foo"}, Object: sequential},
	}
	assert.Equal(t, "out.Invalids++", sequential.InvalidsIncrement("out"))
	assert.Equal(t, "fs.Invalids++", sequential.InvalidsIncrement("fs"))

	// concurrent: at least one resolver field
	obj := &Object{Definition: &ast.Definition{Name: "User"}}
	obj.Fields = []*Field{
		{
			FieldDefinition: &ast.FieldDefinition{Name: "name"},
			IsResolver:      true,
			Object:          obj,
		},
	}
	assert.Equal(t, "atomic.AddUint32(&out.Invalids, 1)", obj.InvalidsIncrement("out"))
	assert.Equal(t, "atomic.AddUint32(&fs.Invalids, 1)", obj.InvalidsIncrement("fs"))
}

func TestObjectInvalidsIncrement_DisableConcurrency(t *testing.T) {
	// DisableConcurrency=true makes IsConcurrent() false even with resolver fields
	obj := &Object{
		Definition:         &ast.Definition{Name: "Mutation"},
		DisableConcurrency: true,
	}
	obj.Fields = []*Field{
		{
			FieldDefinition: &ast.FieldDefinition{Name: "createUser"},
			IsResolver:      true,
			Object:          obj,
		},
	}
	assert.Equal(t, "out.Invalids++", obj.InvalidsIncrement("out"))
}

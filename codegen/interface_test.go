package codegen

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShapes(t *testing.T) {
	err := generate("shapes", `
			type Query {
				shapes: [Shape]
			}
			interface Shape {
				area: Float
			}
			type Circle implements Shape {
				radius: Float
				area: Float
			}
			type Rectangle implements Shape {
				length: Float
				width: Float
				area: Float
			}
			union ShapeUnion = Circle | Rectangle
	`, TypeMap{
		"Shape":      {Model: "github.com/99designs/gqlgen/codegen/testserver.Shape"},
		"ShapeUnion": {Model: "github.com/99designs/gqlgen/codegen/testserver.ShapeUnion"},
		"Circle":     {Model: "github.com/99designs/gqlgen/codegen/testserver.Circle"},
		"Rectangle":  {Model: "github.com/99designs/gqlgen/codegen/testserver.Rectangle"},
	})

	require.NoError(t, err)

}

func TestInterfaceUnexportedMethods(t *testing.T) {
	var buf bytes.Buffer
	stdErrLog.SetOutput(&buf)
	err := generate("interface_unexported_methods", `
			schema {
				query: Query
			}
			
			type Query {
			    entities: [Entity!]
			}
			
			interface Entity {
			    id: ID!
			}
			
			type Foo implements Entity {
			    id: ID!
				x: String!
			}
			
			type Bar implements Entity {
			    id: ID!
				y: String!
			}
	`, TypeMap{
		"Bar":    {Model: "github.com/99designs/gqlgen/codegen/testserver.IfaceBar"},
		"Entity": {Model: "github.com/99designs/gqlgen/codegen/testserver.IfaceEntity"},
		"Foo":    {Model: "github.com/99designs/gqlgen/codegen/testserver.IfaceFoo"},
	})
	require.NoError(t, err)
	assert.Equal(t, "", buf.String(), "using an interface with unexported methods should not emit warnings")
}

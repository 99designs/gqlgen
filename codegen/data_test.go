package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
)

func TestData_Directives(t *testing.T) {
	d := Data{
		Config: &config.Config{
			Sources: []*ast.Source{
				{
					Name: "schema.graphql",
				},
			},
		},
		AllDirectives: DirectiveList{
			"includeDirective": {
				DirectiveDefinition: &ast.DirectiveDefinition{
					Name: "includeDirective",
					Position: &ast.Position{
						Src: &ast.Source{
							Name: "schema.graphql",
						},
					},
				},
				Name: "includeDirective",
				Args: nil,
				DirectiveConfig: config.DirectiveConfig{
					SkipRuntime: false,
				},
			},
			"excludeDirective": {
				DirectiveDefinition: &ast.DirectiveDefinition{
					Name: "excludeDirective",
					Position: &ast.Position{
						Src: &ast.Source{
							Name: "anothersource.graphql",
						},
					},
				},
				Name: "excludeDirective",
				Args: nil,
				DirectiveConfig: config.DirectiveConfig{
					SkipRuntime: false,
				},
			},
		},
	}

	expected := DirectiveList{
		"includeDirective": {
			DirectiveDefinition: &ast.DirectiveDefinition{
				Name: "includeDirective",
				Position: &ast.Position{
					Src: &ast.Source{
						Name: "schema.graphql",
					},
				},
			},
			Name: "includeDirective",
			Args: nil,
			DirectiveConfig: config.DirectiveConfig{
				SkipRuntime: false,
			},
		},
	}

	assert.Equal(t, expected, d.Directives())
}

func TestUniqueChildFieldTypes(t *testing.T) {
	userDef := &ast.Definition{
		Kind:   ast.Object,
		Name:   "User",
		Fields: ast.FieldList{{Name: "id"}, {Name: "name"}},
	}
	stringDef := &ast.Definition{
		Kind: ast.Scalar,
		Name: "String",
	}
	postDef := &ast.Definition{
		Kind:   ast.Object,
		Name:   "Post",
		Fields: ast.FieldList{{Name: "title"}},
	}
	emptyObjDef := &ast.Definition{
		Kind: ast.Object,
		Name: "Empty",
	}

	d := Data{
		Objects: Objects{
			{
				Fields: []*Field{
					{TypeReference: &config.TypeReference{Definition: userDef}},
					// scalar, should be excluded
					{TypeReference: &config.TypeReference{Definition: stringDef}},
					// duplicate, should be deduped
					{TypeReference: &config.TypeReference{Definition: userDef}},
					{TypeReference: &config.TypeReference{Definition: postDef}},
					// no fields, should be excluded
					{TypeReference: &config.TypeReference{
						Definition: emptyObjDef,
					}},
					// nil ref, should be skipped
					{TypeReference: nil},
				},
			},
		},
	}

	result := d.UniqueChildFieldTypes()

	// Should be sorted alphabetically: Post, User
	assert.Len(t, result, 2)
	assert.Equal(t, "Post", result[0].TypeName)
	assert.Equal(t, postDef, result[0].Definition)
	assert.Equal(t, "User", result[1].TypeName)
	assert.Equal(t, userDef, result[1].Definition)
}

func TestUniqueChildFieldTypes_Empty(t *testing.T) {
	d := Data{}
	assert.Empty(t, d.UniqueChildFieldTypes())
}

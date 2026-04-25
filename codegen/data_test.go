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

func TestDataECHelpers(t *testing.T) {
	receiver := &Data{Config: &config.Config{UseFunctionSyntaxForExecutionContext: false}}
	assert.Equal(t, "(ec *executionContext) ", receiver.FuncReceiver())
	assert.Empty(t, receiver.ECFuncParam())
	assert.Equal(t, "ec.", receiver.ECDot())
	assert.Empty(t, receiver.ECArg())

	function := &Data{Config: &config.Config{UseFunctionSyntaxForExecutionContext: true}}
	assert.Empty(t, function.FuncReceiver())
	assert.Equal(t, "ec *executionContext, ", function.ECFuncParam())
	assert.Empty(t, function.ECDot())
	assert.Equal(t, "ec, ", function.ECArg())
}

// implDirectivesFieldStub satisfies ImplDirectivesField for testing purposes.
type implDirectivesFieldStub struct{ zeroVal string }

func (s *implDirectivesFieldStub) DirectiveObjName() string     { return "obj" }
func (s *implDirectivesFieldStub) ImplDirectives() []*Directive { return nil }
func (s *implDirectivesFieldStub) ZeroVal() string              { return s.zeroVal }

func TestImplDirectivesContext_ErrReturn(t *testing.T) {
	// ErrWrap=false: declares the zero value then returns with plain error.
	plain := ImplDirectivesContext{
		ErrWrap: false,
		ErrVal:  "zeroVal",
		Field:   &implDirectivesFieldStub{zeroVal: "var zeroVal string"},
		Data:    &Data{Config: &config.Config{}},
	}
	assert.Equal(t, "var zeroVal string\nreturn zeroVal, err", plain.ErrReturn("err"))
	assert.Equal(t,
		"var zeroVal string\nreturn zeroVal, errors.New(\"not found\")",
		plain.ErrReturn(`errors.New("not found")`),
	)

	// ErrWrap=true: wraps error in graphql.ErrorOnPath, no zero-value declaration.
	wrapped := ImplDirectivesContext{
		ErrWrap: true,
		ErrVal:  "it",
		Field:   &inputObjectImplDirectivesField{},
		Data:    &Data{Config: &config.Config{}},
	}
	assert.Equal(t, "return it, graphql.ErrorOnPath(ctx, err)", wrapped.ErrReturn("err"))
	assert.Equal(t,
		`return it, graphql.ErrorOnPath(ctx, errors.New("directive foo is not implemented"))`,
		wrapped.ErrReturn(`errors.New("directive foo is not implemented")`),
	)
}

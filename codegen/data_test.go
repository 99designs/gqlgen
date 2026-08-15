package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func batchTestObject(name string, root bool, batchFields ...string) *Object {
	obj := &Object{
		Definition: &ast.Definition{Name: name},
		Root:       root,
	}
	for _, f := range batchFields {
		obj.Fields = append(obj.Fields, &Field{
			FieldDefinition: &ast.FieldDefinition{Name: f},
			Batch:           true,
		})
	}
	return obj
}

func TestData_HasBatchFields(t *testing.T) {
	d := Data{
		Objects: Objects{
			batchTestObject("Query", true, "users"),
			batchTestObject("User", false, "posts"),
			batchTestObject("Post", false),
		},
	}
	// Plain field, no batch resolver.
	d.Objects[2].Fields = append(d.Objects[2].Fields, &Field{
		FieldDefinition: &ast.FieldDefinition{Name: "title"},
	})

	cases := map[string]struct {
		typeName string
		want     bool
	}{
		"type with a batch field":    {typeName: "User", want: true},
		"type with only plain field": {typeName: "Post", want: false},
		"root type is never batched": {typeName: "Query", want: false},
		"unknown type":               {typeName: "Missing", want: false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, d.HasBatchFields(tc.typeName))
		})
	}

	// A root-only batch field must not make the schema look batched, otherwise every
	// marshaler in the schema registers parents nothing will ever read.
	rootOnly := Data{Objects: Objects{batchTestObject("Query", true, "users")}}
	assert.False(t, rootOnly.HasBatchResolverFields())
	assert.True(t, d.HasBatchResolverFields())
}

func TestData_BatchImplementors(t *testing.T) {
	d := Data{
		Objects: Objects{
			batchTestObject("Dog", false, "name"),
			batchTestObject("Cat", false),
		},
		Interfaces: map[string]*Interface{
			"Animal": {
				Definition: &ast.Definition{Name: "Animal"},
				Implementors: []InterfaceImplementor{
					{Definition: &ast.Definition{Name: "Dog"}},
					{Definition: &ast.Definition{Name: "Cat"}},
				},
			},
			"Empty": {
				Definition:   &ast.Definition{Name: "Empty"},
				Implementors: []InterfaceImplementor{{Definition: &ast.Definition{Name: "Cat"}}},
			},
		},
	}

	got := d.BatchImplementors("Animal")
	require.Len(t, got, 1)
	assert.Equal(t, "Dog", got[0].Name)

	// No implementor needs grouping, so the marshaler must emit nothing at all: an
	// empty type switch would not compile.
	assert.Empty(t, d.BatchImplementors("Empty"))
	assert.Empty(t, d.BatchImplementors("NotAnInterface"))
}

package codegen

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	ast2 "github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/internal/code"
)

func TestFindField(t *testing.T) {
	input := `
package test

type Std struct {
	Name string
	Value int
}
type Anon struct {
	Name string
	Tags
}
type Tags struct {
	Bar string ` + "`" + `gqlgen:"foo"` + "`" + `
	Foo int    ` + "`" + `gqlgen:"bar"` + "`" + `
}
type Amb struct {
	Bar string ` + "`" + `gqlgen:"foo"` + "`" + `
	Foo int    ` + "`" + `gqlgen:"foo"` + "`" + `
}
type Embed struct {
	Std
	Test string
}
`
	scope, err := parseScope(input, "test")
	require.NoError(t, err)

	std := scope.Lookup("Std").Type().(*types.Named)
	anon := scope.Lookup("Anon").Type().(*types.Named)
	tags := scope.Lookup("Tags").Type().(*types.Named)
	amb := scope.Lookup("Amb").Type().(*types.Named)
	embed := scope.Lookup("Embed").Type().(*types.Named)

	tests := []struct {
		Name        string
		Named       *types.Named
		Field       string
		Tag         string
		Expected    string
		ShouldError bool
	}{
		{"Finds a field by name with no tag", std, "name", "", "Name", false},
		{
			"Finds a field by name when passed tag but tag not used",
			std,
			"name",
			"gqlgen",
			"Name",
			false,
		},
		{"Ignores tags when not passed a tag", tags, "foo", "", "Foo", false},
		{
			"Picks field with tag over field name when passed a tag",
			tags,
			"foo",
			"gqlgen",
			"Bar",
			false,
		},
		{"Errors when ambiguous", amb, "foo", "gqlgen", "", true},
		{"Finds a field that is in embedded struct", anon, "bar", "", "Bar", false},
		{"Finds field that is not in embedded struct", embed, "test", "", "Test", false},
	}

	for _, tt := range tests {
		b := builder{Config: &config.Config{StructTag: tt.Tag}}
		target, err := b.findBindTarget(tt.Named, tt.Field, false)
		if tt.ShouldError {
			require.Nil(t, target, tt.Name)
			require.Error(t, err, tt.Name)
		} else {
			require.NoError(t, err, tt.Name)
			require.Equal(t, tt.Expected, target.Name(), tt.Name)
		}
	}
}

func parseScope(input any, packageName string) (*types.Scope, error) {
	// test setup to parse the types
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", input, 0)
	if err != nil {
		return nil, err
	}

	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check(packageName, fset, []*ast.File{f}, nil)
	if err != nil {
		return nil, err
	}

	return pkg.Scope(), nil
}

func TestEqualFieldName(t *testing.T) {
	tt := []struct {
		Name     string
		Source   string
		Target   string
		Expected bool
	}{
		{Name: "words with same case", Source: "test", Target: "test", Expected: true},
		{Name: "words different case", Source: "test", Target: "tEsT", Expected: true},
		{Name: "different words", Source: "foo", Target: "bar", Expected: false},
		{Name: "separated with underscore", Source: "the_test", Target: "TheTest", Expected: true},
		{Name: "empty values", Source: "", Target: "", Expected: true},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			result := equalFieldName(tc.Source, tc.Target)
			require.Equal(t, tc.Expected, result)
		})
	}
}

func TestField_Batch(t *testing.T) {
	t.Run("Batch flag defaults to false", func(t *testing.T) {
		f := Field{}
		require.False(t, f.Batch)
		require.False(t, f.IsBatch())
	})

	t.Run("Batch flag can be set", func(t *testing.T) {
		f := Field{Batch: true}
		require.True(t, f.Batch)
		require.True(t, f.IsBatch())
	})
}

func TestField_BatchRootFieldUnsupported(t *testing.T) {
	cfg := &config.Config{
		Exec: config.ExecConfig{
			Layout:   config.ExecLayoutSingleFile,
			Filename: "generated.go",
			Package:  "generated",
		},
		Models: config.TypeMap{
			"Query": {
				Fields: map[string]config.TypeMapField{
					"version": {Batch: true},
				},
			},
			"Boolean": {
				Model: config.StringList{"github.com/99designs/gqlgen/graphql.Boolean"},
			},
			"Float": {
				Model: config.StringList{"github.com/99designs/gqlgen/graphql.Float"},
			},
			"ID": {
				Model: config.StringList{"github.com/99designs/gqlgen/graphql.ID"},
			},
			"Int": {
				Model: config.StringList{"github.com/99designs/gqlgen/graphql.Int"},
			},
			"String": {
				Model: config.StringList{"github.com/99designs/gqlgen/graphql.String"},
			},
		},
		Directives: map[string]config.DirectiveConfig{},
		Packages:   code.NewPackages(),
	}
	cfg.Schema = gqlparser.MustLoadSchema(&ast2.Source{
		Name: "schema.graphql",
		Input: `
			schema { query: Query }
			type Query { version: String }
		`,
	})

	b := builder{
		Config: cfg,
		Schema: cfg.Schema,
	}
	b.Binder = b.Config.NewBinder()
	var err error
	b.Directives, err = b.buildDirectives()
	require.NoError(t, err)

	_, err = b.buildObject(cfg.Schema.Query)
	require.Error(t, err)
	require.Contains(t, err.Error(), "batch resolver is not supported for root field Query.version")
}

func TestField_CallArgs(t *testing.T) {
	tt := []struct {
		Name string
		Field
		Expected string
	}{
		{
			Name: "Field with method that has context, and three args (string, interface, named interface)",
			Field: Field{
				MethodHasContext: true,
				Args: []*FieldArgument{
					{
						ArgumentDefinition: &ast2.ArgumentDefinition{
							Name: "test",
						},
						TypeReference: &config.TypeReference{
							GO: (&types.Interface{}).Complete(),
						},
					},
					{
						ArgumentDefinition: &ast2.ArgumentDefinition{
							Name: "test2",
						},
						TypeReference: &config.TypeReference{
							GO: types.NewNamed(
								types.NewTypeName(token.NoPos, nil, "TestInterface", nil),
								(&types.Interface{}).Complete(),
								nil,
							),
						},
					},
					{
						ArgumentDefinition: &ast2.ArgumentDefinition{
							Name: "test3",
						},
						TypeReference: &config.TypeReference{
							GO: types.Typ[types.String],
						},
					},
				},
			},
			Expected: `ctx, ` + `
				func () any {
					if fc.Args["test"] == nil {
						return nil
					}
					return fc.Args["test"].(any)
				}(), fc.Args["test2"].(TestInterface), fc.Args["test3"].(string)`,
		},
		{
			Name: "Resolver field that isn't root object with single int argument",
			Field: Field{
				Object: &Object{
					Root: false,
				},
				IsResolver: true,
				Args: []*FieldArgument{
					{
						ArgumentDefinition: &ast2.ArgumentDefinition{
							Name: "test",
						},
						TypeReference: &config.TypeReference{
							GO: types.Typ[types.Int],
						},
					},
				},
			},
			Expected: `ctx, obj, fc.Args["test"].(int)`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			require.Equal(t, tc.Expected, tc.CallArgs())
		})
	}
}

func TestField_BatchCallArgs(t *testing.T) {
	tt := []struct {
		Name     string
		Field    Field
		Expected string
	}{
		{
			Name: "Batch args with single int argument",
			Field: Field{
				Args: []*FieldArgument{
					{
						ArgumentDefinition: &ast2.ArgumentDefinition{
							Name: "test",
						},
						TypeReference: &config.TypeReference{
							GO: types.Typ[types.Int],
						},
					},
				},
			},
			Expected: `ctx, parents, fc.Args["test"].(int)`,
		},
		{
			Name: "Batch args with empty interface and string",
			Field: Field{
				Args: []*FieldArgument{
					{
						ArgumentDefinition: &ast2.ArgumentDefinition{
							Name: "test",
						},
						TypeReference: &config.TypeReference{
							GO: (&types.Interface{}).Complete(),
						},
					},
					{
						ArgumentDefinition: &ast2.ArgumentDefinition{
							Name: "test2",
						},
						TypeReference: &config.TypeReference{
							GO: types.Typ[types.String],
						},
					},
				},
			},
			Expected: `ctx, parents, ` + `
				func () any {
					if fc.Args["test"] == nil {
						return nil
					}
					return fc.Args["test"].(any)
				}(), fc.Args["test2"].(string)`,
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			require.Equal(t, tc.Expected, tc.Field.BatchCallArgs("parents"))
		})
	}
}

func TestField_ShortBatchResolverDeclaration(t *testing.T) {
	f := Field{
		FieldDefinition: &ast2.FieldDefinition{
			Name: "value",
		},
		Object: &Object{
			Definition: &ast2.Definition{
				Name: "User",
			},
			Type: types.Typ[types.Int],
		},
		TypeReference: &config.TypeReference{
			GO: types.Typ[types.String],
		},
		Args: []*FieldArgument{
			{
				ArgumentDefinition: &ast2.ArgumentDefinition{
					Name: "limit",
				},
				VarName: "limit",
				TypeReference: &config.TypeReference{
					GO: types.Typ[types.Int],
				},
			},
		},
	}

	require.Equal(
		t,
		"(ctx context.Context, objs []*int, limit int) ([]graphql.BatchResult[string])",
		f.ShortBatchResolverDeclaration(),
	)
}

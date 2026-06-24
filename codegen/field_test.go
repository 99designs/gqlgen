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

func TestField_StreamResolverShape(t *testing.T) {
	newField := func(stream, marked bool) *Field {
		var directives ast2.DirectiveList
		if marked {
			directives = ast2.DirectiveList{{Name: config.DirSubscriptionContext}}
		}
		return &Field{
			FieldDefinition: &ast2.FieldDefinition{Directives: directives},
			Object:          &Object{Stream: stream},
		}
	}

	tests := map[string]struct {
		field          *Field
		wantReturnType string
		wantResolveFn  string
	}{
		"non-stream field": {
			field:          newField(false, false),
			wantReturnType: "graphql.Marshaler",
			wantResolveFn:  "ResolveField",
		},
		"stream field without @subscriptionContext": {
			field:          newField(true, false),
			wantReturnType: "func(ctx context.Context) graphql.Marshaler",
			wantResolveFn:  "ResolveFieldStream",
		},
		"stream field with @subscriptionContext": {
			field:          newField(true, true),
			wantReturnType: "func(ctx context.Context) (context.Context, graphql.Marshaler)",
			wantResolveFn:  "ResolveFieldStreamWithEventContext",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.wantReturnType, tt.field.MarshalerReturnType())
			require.Equal(t, tt.wantResolveFn, tt.field.ResolveFieldFunc())
		})
	}
}

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

func TestField_BatchRootField(t *testing.T) {
	baseModels := config.TypeMap{
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
	}

	buildQuery := func(t *testing.T, cfg *config.Config) *Object {
		t.Helper()
		b := builder{
			Config: cfg,
			Schema: cfg.Schema,
		}
		b.Binder = b.Config.NewBinder()
		var err error
		b.Directives, err = b.buildDirectives()
		require.NoError(t, err)

		obj, err := b.buildObject(cfg.Schema.Query)
		require.NoError(t, err)
		return obj
	}

	t.Run("global batch skips root fields", func(t *testing.T) {
		cfg := &config.Config{
			Resolver: config.ResolverConfig{
				Batch: config.ResolverBatchConfig{Enabled: true},
			},
			Exec: config.ExecConfig{
				Layout:   config.ExecLayoutSingleFile,
				Filename: "generated.go",
				Package:  "generated",
			},
			Models:     baseModels,
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

		obj := buildQuery(t, cfg)
		require.Len(t, obj.Fields, 1)
		require.False(t, obj.Fields[0].Batch)
	})

	t.Run("explicit batch on root field is rejected", func(t *testing.T) {
		batchTrue := true
		models := make(config.TypeMap)
		for k, v := range baseModels {
			models[k] = v
		}
		models["Query"] = config.TypeMapEntry{
			Fields: map[string]config.TypeMapField{
				"version": {Batch: &batchTrue},
			},
		}

		cfg := &config.Config{
			Resolver: config.ResolverConfig{
				Batch: config.ResolverBatchConfig{Enabled: true},
			},
			Exec: config.ExecConfig{
				Layout:   config.ExecLayoutSingleFile,
				Filename: "generated.go",
				Package:  "generated",
			},
			Models:     models,
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
		require.Contains(t, err.Error(), "batch resolver is not supported for field Query.version")
	})
}

func TestField_BatchGlobalWithQueryAndUser(t *testing.T) {
	cfg := &config.Config{
		Resolver: config.ResolverConfig{
			Batch: config.ResolverBatchConfig{Enabled: true},
		},
		Exec: config.ExecConfig{
			Layout:   config.ExecLayoutSingleFile,
			Filename: "generated.go",
			Package:  "generated",
		},
		Models: config.TypeMap{
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
			"User": {
				Model: config.StringList{"map[string]interface{}"},
				Fields: map[string]config.TypeMapField{
					"posts": {Resolver: true},
				},
			},
			"Post": {
				Model: config.StringList{"map[string]interface{}"},
			},
		},
		Directives: map[string]config.DirectiveConfig{},
		Packages:   code.NewPackages(),
	}
	cfg.Schema = gqlparser.MustLoadSchema(&ast2.Source{
		Name: "schema.graphql",
		Input: `
			schema { query: Query }
			type Query { version: String users: [User!]! }
			type User { id: ID! posts: [Post!]! }
			type Post { id: ID! }
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

	queryObj, err := b.buildObject(cfg.Schema.Query)
	require.NoError(t, err)
	require.False(t, fieldByName(queryObj.Fields, "version").Batch)
	require.False(t, fieldByName(queryObj.Fields, "users").Batch)

	userObj, err := b.buildObject(cfg.Schema.Types["User"])
	require.NoError(t, err)
	require.True(t, fieldByName(userObj.Fields, "posts").Batch)
	require.False(t, fieldByName(userObj.Fields, "id").Batch)
	require.False(t, fieldByName(userObj.Fields, "id").IsResolver)
}

func TestField_BatchEntityType(t *testing.T) {
	baseModels := config.TypeMap{
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
		"Entity": {
			Model: config.StringList{"map[string]interface{}"},
			Fields: map[string]config.TypeMapField{
				"id": {Resolver: true},
			},
		},
	}

	buildEntity := func(t *testing.T, cfg *config.Config) *Object {
		t.Helper()
		b := builder{
			Config: cfg,
			Schema: cfg.Schema,
		}
		b.Binder = b.Config.NewBinder()
		var err error
		b.Directives, err = b.buildDirectives()
		require.NoError(t, err)

		obj, err := b.buildObject(cfg.Schema.Types["Entity"])
		require.NoError(t, err)
		return obj
	}

	t.Run("global batch applies to Entity without federation", func(t *testing.T) {
		cfg := &config.Config{
			Resolver: config.ResolverConfig{
				Batch: config.ResolverBatchConfig{Enabled: true},
			},
			Exec: config.ExecConfig{
				Layout:   config.ExecLayoutSingleFile,
				Filename: "generated.go",
				Package:  "generated",
			},
			Models:     baseModels,
			Directives: map[string]config.DirectiveConfig{},
			Packages:   code.NewPackages(),
		}
		cfg.Schema = gqlparser.MustLoadSchema(&ast2.Source{
			Name: "schema.graphql",
			Input: `
				schema { query: Query }
				type Query { _: Boolean }
				type Entity { id: ID! }
			`,
		})

		obj := buildEntity(t, cfg)
		require.True(t, fieldByName(obj.Fields, "id").Batch)
	})

	t.Run("global batch skips federation Entity", func(t *testing.T) {
		cfg := &config.Config{
			Resolver: config.ResolverConfig{
				Batch: config.ResolverBatchConfig{Enabled: true},
			},
			Exec: config.ExecConfig{
				Layout:   config.ExecLayoutSingleFile,
				Filename: "generated.go",
				Package:  "generated",
			},
			Federation: config.PackageConfig{
				Filename: "graph/federation.go",
				Package:  "graph",
			},
			Models:     baseModels,
			Directives: map[string]config.DirectiveConfig{},
			Packages:   code.NewPackages(),
		}
		cfg.Schema = gqlparser.MustLoadSchema(&ast2.Source{
			Name: "schema.graphql",
			Input: `
				schema { query: Query }
				type Query { _: Boolean }
				type Entity { id: ID! }
			`,
		})

		obj := buildEntity(t, cfg)
		require.False(t, fieldByName(obj.Fields, "id").Batch)
	})
}

func fieldByName(fields []*Field, name string) *Field {
	for _, f := range fields {
		if f.Name == name {
			return f
		}
	}
	return nil
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
			require.Equal(t, tc.Expected, tc.Field.BatchCallArgs("parents", ""))
		})
	}
}

func TestField_HasFederationRequiresArg(t *testing.T) {
	f := Field{
		Args: []*FieldArgument{
			{ArgumentDefinition: &ast2.ArgumentDefinition{Name: "_federationRequires"}},
		},
	}
	require.True(t, f.HasFederationRequiresArg())

	f.Args = nil
	require.False(t, f.HasFederationRequiresArg())
}

func TestField_ShortBatchResolverDeclaration_FederationRequires(t *testing.T) {
	mapType := types.NewMap(types.Typ[types.String], types.NewInterfaceType(nil, nil).Complete())
	f := Field{
		FieldDefinition: &ast2.FieldDefinition{Name: "size"},
		Object: &Object{
			Definition: &ast2.Definition{Name: "Product"},
			Type: types.NewPointer(
				types.NewNamed(
					types.NewTypeName(0, nil, "Product", nil),
					types.NewStruct(nil, nil),
					nil,
				),
			),
		},
		TypeReference: &config.TypeReference{GO: types.Typ[types.Int]},
		Args: []*FieldArgument{
			{
				ArgumentDefinition: &ast2.ArgumentDefinition{Name: "_federationRequires"},
				VarName:            "federationRequires",
				TypeReference:      &config.TypeReference{GO: mapType},
			},
		},
	}

	require.Equal(
		t,
		"(ctx context.Context, objs []*Product, federationRequires []map[string]interface{}) ([]int, error)",
		f.ShortBatchResolverDeclaration(),
	)
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
		"(ctx context.Context, objs []*int, limit int) ([]string, error)",
		f.ShortBatchResolverDeclaration(),
	)
}

func TestField_ChildFieldContextTypeName(t *testing.T) {
	t.Run("returns definition name", func(t *testing.T) {
		f := Field{
			TypeReference: &config.TypeReference{
				Definition: &ast2.Definition{Name: "User"},
			},
		}
		require.Equal(t, "User", f.ChildFieldContextTypeName())
	})

	t.Run("nil TypeReference", func(t *testing.T) {
		f := Field{TypeReference: nil}
		require.Empty(t, f.ChildFieldContextTypeName())
	})

	t.Run("nil Definition", func(t *testing.T) {
		f := Field{TypeReference: &config.TypeReference{Definition: nil}}
		require.Empty(t, f.ChildFieldContextTypeName())
	})
}

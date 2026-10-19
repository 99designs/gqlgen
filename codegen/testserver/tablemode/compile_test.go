//go:build !race

// The tests of this file and determinism_test.go run the generator dozens of times. It
// starts no goroutines, so the race detector finds nothing in them while it makes them
// several times slower; CI runs them without -race in a step of their own. They generate
// in the exec mode that the test binary is built for, execMode: CI runs them both without
// and with the gqlgen_table tag, so that they check both modes.

package tablemode_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"golang.org/x/tools/go/packages"

	"github.com/99designs/gqlgen/api"
	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
)

// TestTableModeCompiles generates, in each exec mode, schemas whose bindings the
// functions mode compiles, and type-checks the generated packages: table mode must compile
// what the functions mode does. In the schemas, a field of an object is bound to a Go
// type that its marshaler does not take, which the functions mode returns as any, the
// functions of types of the schema have the names that table mode gives its own, the
// configuration enables batch resolvers that the schema turns off or does not have, an
// interface is bound to an implementation and to another interface, Go types are in
// packages called schema and data, and a scalar is bound to a marshal function that
// returns another type than graphql.Marshaler.
func TestTableModeCompiles(t *testing.T) {
	for _, name := range []string{
		"omittable", "mapmodel", "names", "batchoff", "ifacemodel", "shadowed", "funcpair",
		"fedentity",
	} {
		t.Run(name, func(t *testing.T) {
			configPath, err := filepath.Abs(filepath.Join("compile", name, "gqlgen.yml"))
			require.NoError(t, err)
			dir := generateDir(t, configPath)
			require.Empty(t, typeErrors(t, dir), "exec mode %s", execMode)
		})
	}
}

// TestTableModeFailsToCompile generates, in each exec mode, schemas whose bindings the
// functions mode does not compile, and requires type errors in table mode too, rather than
// code that fails when it runs: fields and arguments take the second Go type of an object
// and an input bound to two, a map implements an interface by a value receiver, and an
// input unmarshals itself without marshaling itself.
func TestTableModeFailsToCompile(t *testing.T) {
	for _, name := range []string{
		"multimodel", "nilableimpl", "unmarshalonly", "ptrmarshal", "badunmarshal", "shadowv",
	} {
		t.Run(name, func(t *testing.T) {
			configPath, err := filepath.Abs(filepath.Join("compile", name, "gqlgen.yml"))
			require.NoError(t, err)
			dir := generateDir(t, configPath)
			require.NotEmpty(t, typeErrors(t, dir), "exec mode %s", execMode)
		})
	}
}

// addedDirective declares @added on FIELD_DEFINITION in a source of its own, which the
// configuration does not hold, and puts it on Query.user.
type addedDirective struct{}

func (addedDirective) Name() string { return "added" }

func (addedDirective) MutateSchema(s *ast.Schema) error {
	pos := &ast.Position{Src: &ast.Source{Name: "plugin.graphql"}}
	def := &ast.DirectiveDefinition{
		Name:      "added",
		Locations: []ast.DirectiveLocation{ast.LocationFieldDefinition},
		Position:  pos,
	}
	s.Directives[def.Name] = def
	user := s.Query.Fields.ForName("user")
	user.Directives = append(
		user.Directives,
		&ast.Directive{
			Name:       def.Name,
			Definition: def,
			Location:   ast.LocationFieldDefinition,
			Position:   pos,
		},
	)
	return nil
}

// batchID makes User.id a batch resolver in the configuration.
type batchID struct{}

func (batchID) Name() string { return "batch" }

func (batchID) MutateConfig(cfg *config.Config) error {
	m := cfg.Models["User"]
	if m.Fields == nil {
		m.Fields = map[string]config.TypeMapField{}
	}
	m.Fields["id"] = config.TypeMapField{Resolver: true, Batch: new(true)}
	cfg.Models["User"] = m
	return nil
}

// guardedData puts @guard on Query.guarded in the data of the generated code, as a plugin
// that implements directives would.
type guardedData struct{}

func (guardedData) Name() string { return "guarded" }

func (guardedData) GenerateCode(data *codegen.Data) error {
	for _, f := range data.QueryRoot.Fields {
		if f.Name == "guarded" {
			f.Directives = append(f.Directives, data.AllDirectives["guard"])
		}
	}
	return nil
}

// sequentialQuery resolves the fields of the query type one after another in the data of
// the generated code.
type sequentialQuery struct{}

func (sequentialQuery) Name() string { return "sequential" }

func (sequentialQuery) GenerateCode(data *codegen.Data) error {
	data.QueryRoot.DisableConcurrency = true
	return nil
}

// nullArgs runs the directive of the argument of Query.noted even when the argument is
// absent, in the data of the generated code.
type nullArgs struct{}

func (nullArgs) Name() string { return "null-args" }

func (nullArgs) GenerateCode(data *codegen.Data) error {
	for _, f := range data.QueryRoot.Fields {
		if f.Name == "noted" {
			f.Args[0].CallArgumentDirectivesWithNull = true
		}
	}
	return nil
}

// nullConfig sets call_argument_directives_with_null in the configuration of the data of
// the generated code.
type nullConfig struct{}

func (nullConfig) Name() string { return "null-config" }

func (nullConfig) GenerateCode(data *codegen.Data) error {
	data.Config.CallArgumentDirectivesWithNull = true
	return nil
}

// guardedSource puts @guard on Query.guarded in the source that the generated code holds.
type guardedSource struct{}

func (guardedSource) Name() string { return "guarded-source" }

func (guardedSource) GenerateCode(data *codegen.Data) error {
	for i := range data.AugmentedSources {
		src := &data.AugmentedSources[i]
		src.Source = strings.Replace(src.Source, "guarded: Int", "guarded: Int @guard", 1)
	}
	return nil
}

// runtimeDirective declares @generatorOnly, which skip_runtime leaves to the generator,
// among the directives that the generated code implements.
type runtimeDirective struct{}

func (runtimeDirective) Name() string { return "runtime-directive" }

func (runtimeDirective) GenerateCode(data *codegen.Data) error {
	data.AllDirectives["generatorOnly"] = &codegen.Directive{
		DirectiveDefinition: data.Schema.Directives["generatorOnly"],
		Name:                "generatorOnly",
	}
	return nil
}

// batchData resolves User.id with a batch resolver in the data of the generated code.
type batchData struct{}

func (batchData) Name() string { return "batch-data" }

func (batchData) GenerateCode(data *codegen.Data) error {
	for _, f := range data.Objects.ByName("User").Fields {
		if f.Name == "id" {
			f.IsResolver = true
			f.Batch = true
		}
	}
	return nil
}

// Table mode fails the generation, with an error that names what it does not support,
// where a plugin makes the functions mode run what the tables cannot: a directive that
// the configuration does not declare, which the functions mode does not compile either,
// and changes to the data of the generated code that the runtime reads from the schema
// instead.
func TestTableModeRejectsPluginChanges(t *testing.T) {
	requireTableMode(t)
	configPath, err := filepath.Abs(filepath.Join("compile", "plugins", "gqlgen.yml"))
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		plugin plugin.Plugin
		err    string
	}{
		{"a directive outside the sources", addedDirective{}, "@added"},
		{"a directive in the data", guardedData{}, "Query.guarded"},
		{"sequential fields in the data", sequentialQuery{}, "type Query"},
		{"directives of absent arguments in the data", nullArgs{}, "Query.noted(x:)"},
		{"directives of absent arguments in the configuration", nullConfig{}, "call_argument_directives_with_null"},
		{"a source in the data", guardedSource{}, "the sources"},
		{"a directive left to the generator in the data", runtimeDirective{}, "directive @generatorOnly"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := generateWith(t, configPath, api.AddPlugin(tc.plugin))
			if err == nil {
				t.Fatalf("the generation succeeds, with type errors %v", typeErrors(t, dir))
			}
			require.ErrorContains(t, err, tc.err)
		})
	}
}

// userResolver resolves User.id with a resolver, as a plugin that implements resolvers
// would.
type userResolver struct{}

func (userResolver) Name() string { return "user-resolver" }

func (userResolver) GenerateCode(data *codegen.Data) error {
	for _, f := range data.Objects.ByName("User").Fields {
		if f.Name == "id" {
			f.IsResolver = true
		}
	}
	return nil
}

// rawDirective declares @onRaw in a source of its own, which the configuration does not
// hold, and puts it on a field of Raw, which unmarshals itself.
type rawDirective struct{}

func (rawDirective) Name() string { return "raw-directive" }

func (rawDirective) MutateSchema(s *ast.Schema) error {
	pos := &ast.Position{Src: &ast.Source{Name: "plugin.graphql"}}
	def := &ast.DirectiveDefinition{
		Name:      "onRaw",
		Locations: []ast.DirectiveLocation{ast.LocationInputFieldDefinition},
		Position:  pos,
	}
	s.Directives[def.Name] = def
	a := s.Types["Raw"].Fields.ForName("a")
	a.Directives = append(a.Directives, &ast.Directive{
		Name:       def.Name,
		Definition: def,
		Location:   ast.LocationInputFieldDefinition,
		Position:   pos,
	})
	return nil
}

// rawDefault changes the default of a field of Raw, which unmarshals itself.
type rawDefault struct{}

func (rawDefault) Name() string { return "raw-default" }

func (rawDefault) GenerateCode(data *codegen.Data) error {
	data.Inputs.ByName("Raw").Fields[0].Default = int64(2)
	return nil
}

// Table mode generates the code where a plugin changes the configuration, such as to make
// a field a batch resolver, or the data of the generated code in what the tables hold,
// such as the resolvers, or in what neither mode runs, such as the fields of an input that
// unmarshals itself.
func TestTableModeFollowsPluginChanges(t *testing.T) {
	configPath, err := filepath.Abs(filepath.Join("compile", "plugins", "gqlgen.yml"))
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		plugin plugin.Plugin
	}{
		{"a batch resolver in the configuration", batchID{}},
		{"a batch resolver in the data", batchData{}},
		{"a resolver", userResolver{}},
		{"an input that unmarshals itself", rawDefault{}},
		{"a directive declared outside the sources on an input that unmarshals itself", rawDirective{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := generateWith(t, configPath, api.AddPlugin(tc.plugin))
			require.NoError(t, err, "exec mode %s", execMode)
			require.Empty(t, typeErrors(t, dir), "exec mode %s", execMode)
		})
	}
}

// pluginCaller writes a file into the package of the executor that unmarshals and marshals
// Hello.name and marshals Query.hello, as a plugin that generates code calling the
// marshalers and unmarshalers of the functions mode by their names would.
type pluginCaller struct{}

func (pluginCaller) Name() string { return "plugin-caller" }

func (pluginCaller) GenerateCode(data *codegen.Data) error {
	name := data.Objects.ByName("Hello").Fields[0].TypeReference
	hello := data.Objects.ByName("Query").Fields[0].TypeReference
	src := "package " + data.Config.Exec.Package + `

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

func callUnmarshaler(ctx context.Context, ec *executionContext, v any) (string, error) {
	return ` + data.ECDot() + name.UnmarshalFunc() + "(ctx, " + data.ECArg() + `v)
}

func callMarshalers(ctx context.Context, ec *executionContext, sel ast.SelectionSet, v *Hello) (graphql.Marshaler, graphql.Marshaler) {
	return ` + data.ECDot() + name.MarshalFunc() + "(ctx, " + data.ECArg() + `sel, v.Name), ` +
		data.ECDot() + hello.MarshalFunc() + "(ctx, " + data.ECArg() + `sel, v)
}
`
	return os.WriteFile(filepath.Join(data.Config.Exec.Dir(), "caller.go"), []byte(src), 0o600)
}

// A plugin that runs after the federation plugin can call marshalers and unmarshalers too,
// as methods of the execution context or as functions that take it.
func TestTableModePluginCalls(t *testing.T) {
	configPath, err := filepath.Abs(filepath.Join("compile", "federation", "gqlgen.yml"))
	require.NoError(t, err)
	for _, functionSyntax := range []bool{false, true} {
		t.Run(fmt.Sprintf("function syntax %v", functionSyntax), func(t *testing.T) {
			syntax := func(cfg *config.Config, _ *[]plugin.Plugin) {
				cfg.UseFunctionSyntaxForExecutionContext = functionSyntax
			}
			dir, err := generateWith(t, configPath, syntax, api.AddPlugin(pluginCaller{}))
			require.NoError(t, err, "exec mode %s", execMode)
			require.FileExists(t, filepath.Join(dir, "caller.go"))
			require.Empty(t, typeErrors(t, dir), "exec mode %s", execMode)
		})
	}
}

// A program that runs gqlgen may change a source in memory, which the generated code
// embeds from its file when the file is in the directory of the code. The functions mode
// generates the code of the changed schema, so table mode links the tables to it.
func TestTableModeLinksChangedSources(t *testing.T) {
	requireTableMode(t)
	configDir, err := filepath.Abs(filepath.Join("compile", "plugins"))
	require.NoError(t, err)
	//nolint:usetesting // the directory is inside the module, as in generateWith
	dir, err := os.MkdirTemp(configDir, "_generated-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(dir)) })
	for _, name := range []string{"gqlgen.yml", "schema.graphql"} {
		b, err := os.ReadFile(filepath.Join(configDir, name))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), b, 0o600))
	}
	t.Chdir(dir)
	cfg, err := config.LoadConfig("gqlgen.yml")
	require.NoError(t, err)
	src := cfg.Sources[0]
	changed := strings.Replace(src.Input, "guarded: Int", "guarded: Int @guard", 1)
	require.NotEqual(t, src.Input, changed)
	src.Input = changed
	cfg.Model.Filename = "models-gen.go"
	cfg.Model.Package = cfg.Exec.Package
	cfg.Resolver = config.ResolverConfig{}
	cfg.SkipValidation = true
	cfg.SkipModTidy = true
	require.NoError(t, api.Generate(cfg))

	generated, err := os.ReadFile("generated.go")
	require.NoError(t, err)
	require.Contains(t, string(generated), "guarded: Int @guard")
}

// typeErrors type-checks the package in dir and returns its errors.
func typeErrors(t *testing.T, dir string) []string {
	t.Helper()
	pkgs, err := packages.Load(&packages.Config{
		Dir:  dir,
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax,
	}, ".")
	require.NoError(t, err)
	var errs []string
	for _, p := range pkgs {
		for _, e := range p.Errors {
			errs = append(errs, e.Error())
		}
	}
	return errs
}

package tablecodegen

import (
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
)

// The calls are those of the marshalers and unmarshalers in the files that a plugin may
// generate into the package of the executor, as methods or as functions, but for the files
// of the executor and for the functions that the files declare.
func TestCollectPluginCalls(t *testing.T) {
	scalar := func(name string, goType types.Type) *config.TypeReference {
		return &config.TypeReference{
			GO:         goType,
			GQL:        ast.NonNullNamedType(name, nil),
			Definition: &ast.Definition{Name: name, Kind: ast.Scalar},
		}
	}
	str := scalar("String", types.Typ[types.String])
	integer := scalar("Int", types.Typ[types.Int])
	id := scalar("ID", types.Typ[types.String])
	object := &config.TypeReference{
		GO:         types.Typ[types.String],
		GQL:        ast.NonNullNamedType("Query", nil),
		Definition: &ast.Definition{Name: "Query", Kind: ast.Object},
	}
	input := &config.TypeReference{
		GO:         types.Typ[types.String],
		GQL:        ast.NonNullNamedType("Filter", nil),
		Definition: &ast.Definition{Name: "Filter", Kind: ast.InputObject},
	}

	dir := t.TempDir()
	write := func(name, src string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600))
	}
	write("federation.go", `package graph

func (ec *executionContext) resolve(ctx context.Context, sel ast.SelectionSet, v any) {
	ec.`+str.UnmarshalFunc()+`(ctx, v)
	`+integer.UnmarshalFunc()+`(ctx, ec, v)
	ec.`+object.MarshalFunc()+`(ctx, sel, "")
	`+str.MarshalFunc()+`(ctx, ec, sel, "")
	ec.`+input.UnmarshalFunc()+`(ctx, v)
	// A field that has the name of an unmarshaler, as those of the tables do, is not a call.
	_ = ec.tables.`+id.UnmarshalFunc()+`.In
}
`)
	write("helpers.go", `package graph

func `+id.UnmarshalFunc()+`(ctx context.Context, ec *executionContext, v any) (string, error) {
	return "", nil
}

func use(ctx context.Context, ec *executionContext) { `+id.UnmarshalFunc()+`(ctx, ec, nil) }
`)
	write("generated.go", `package graph

func (ec *executionContext) f(ctx context.Context) { ec.`+id.UnmarshalFunc()+`(ctx, nil) }
`)
	write("broken.go", "package graph\n\nfunc {")
	write("graph_test.go", `package graph

func g(ctx context.Context, ec *executionContext) { ec.`+id.UnmarshalFunc()+`(ctx, nil) }
`)

	data := &codegen.Data{
		Config: &config.Config{Exec: config.ExecConfig{
			Layout:   config.ExecLayoutSingleFile,
			Filename: filepath.Join(dir, "generated.go"),
		}},
		ReferencedTypes: map[string]*config.TypeReference{
			"String": str, "Int": integer, "ID": id, "Query": object, "Filter": input,
		},
	}
	calls, err := collectPluginCalls(data)
	require.NoError(t, err)
	require.Equal(t, map[string]callForms{
		str.UnmarshalFunc():     {method: true},
		integer.UnmarshalFunc(): {function: true},
		object.MarshalFunc():    {method: true},
		str.MarshalFunc():       {function: true},
		input.UnmarshalFunc():   {method: true},
	}, calls)
}

// A reference is a selector of an execution context, called or as a value, or an
// identifier of the name. A declaration takes away only its own form: a function of the
// name, or a method of the name of executionContext.
func TestCollectPluginCallsReferences(t *testing.T) {
	scalar := func(name string) *config.TypeReference {
		return &config.TypeReference{
			GO:         types.Typ[types.String],
			GQL:        ast.NonNullNamedType(name, nil),
			Definition: &ast.Definition{Name: name, Kind: ast.Scalar},
		}
	}
	a, b, c, d, e := scalar("A"), scalar("B"), scalar("C"), scalar("D"), scalar("E")
	dir := t.TempDir()
	src := `package graph

type other struct{}

// A method of another type with the name of a marshaler is not a declaration of it.
func (other) ` + e.UnmarshalFunc() + `(ctx context.Context, v any) (string, error) { return "", nil }

// A function of the name declares the function form, not the method form.
func ` + d.UnmarshalFunc() + `(ctx context.Context, ec *executionContext, v any) (string, error) {
	return "", nil
}

func plugin(ctx context.Context, ec *executionContext, o other, sel ast.SelectionSet) {
	// Values, not calls.
	f := ec.` + a.MarshalFunc() + `
	g := ` + b.UnmarshalFunc() + `
	// A method of another receiver.
	o.` + c.UnmarshalFunc() + `(ctx, nil)
	// A method of the execution context whose function form the file declares.
	ec.` + d.UnmarshalFunc() + `(ctx, nil)
	o.` + e.UnmarshalFunc() + `(ctx, nil)
	ec.` + e.UnmarshalFunc() + `(ctx, nil)
	func(ctx context.Context, inner *executionContext) { inner.` + c.MarshalFunc() + `(ctx, sel, "") }(ctx, ec)
	_, _ = f, g
}

// Variables declared of the type or with a new one are execution contexts too.
func locals(ctx context.Context, sel ast.SelectionSet) {
	made := newExecutionContext(nil, nil, nil)
	made.` + a.UnmarshalFunc() + `(ctx, nil)
	lit := &executionContext{}
	lit.` + b.MarshalFunc() + `(ctx, sel, "")
	var typed *executionContext
	typed.` + d.MarshalFunc() + `(ctx, sel, "")
	other := struct{}{}
	_ = other
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.go"), []byte(src), 0o600))
	data := &codegen.Data{
		Config: &config.Config{Exec: config.ExecConfig{
			Layout:   config.ExecLayoutSingleFile,
			Filename: filepath.Join(dir, "generated.go"),
		}},
		ReferencedTypes: map[string]*config.TypeReference{"A": a, "B": b, "C": c, "D": d, "E": e},
	}
	calls, err := collectPluginCalls(data)
	require.NoError(t, err)
	require.Equal(t, map[string]callForms{
		a.MarshalFunc():   {method: true},
		a.UnmarshalFunc(): {method: true},
		b.UnmarshalFunc(): {function: true},
		b.MarshalFunc():   {method: true},
		c.MarshalFunc():   {method: true},
		d.UnmarshalFunc(): {method: true},
		d.MarshalFunc():   {method: true},
		e.UnmarshalFunc(): {method: true},
	}, calls)
}

// The marshaler of the subscription type also returns the context of the event with the
// subscription_context_field option, which table mode does not generate, so a call to it
// is left to the compiler.
func TestCollectPluginCallsSubscriptionContext(t *testing.T) {
	sub := &config.TypeReference{
		GO:         types.Typ[types.String],
		GQL:        ast.NonNullNamedType("Subscription", nil),
		Definition: &ast.Definition{Name: "Subscription", Kind: ast.Object},
	}
	dir := t.TempDir()
	src := "package graph\n\nfunc f(ctx context.Context, ec *executionContext, sel ast.SelectionSet) { ec." +
		sub.MarshalFunc() + "(ctx, sel, \"\") }\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.go"), []byte(src), 0o600))
	cfg := &config.Config{Exec: config.ExecConfig{
		Layout:   config.ExecLayoutSingleFile,
		Filename: filepath.Join(dir, "generated.go"),
	}}
	data := &codegen.Data{
		Config:          cfg,
		ReferencedTypes: map[string]*config.TypeReference{"Subscription": sub},
	}
	calls, err := collectPluginCalls(data)
	require.NoError(t, err)
	require.Equal(t, map[string]callForms{sub.MarshalFunc(): {method: true}}, calls)

	cfg.SubscriptionContextField = true
	calls, err = collectPluginCalls(data)
	require.NoError(t, err)
	require.Empty(t, calls)
}

// The files of the follow-schema layout are named after the sources of the types, with
// the prelude of the built-in ones, which are not among the sources of the config.
func TestCollectPluginCallsFollowSchema(t *testing.T) {
	str := &config.TypeReference{
		GO:         types.Typ[types.String],
		GQL:        ast.NonNullNamedType("String", nil),
		Definition: &ast.Definition{Name: "String", Kind: ast.Scalar},
	}
	dir := t.TempDir()
	call := "package graph\n\nfunc f(ctx context.Context, ec *executionContext) { ec." +
		str.UnmarshalFunc() + "(ctx, nil) }\n"
	for _, name := range []string{"schema.generated.go", "prelude.generated.go", "root_.generated.go"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(call), 0o600))
	}
	schema := &ast.Source{Name: "graph/schema.graphqls"}
	prelude := &ast.Source{Name: "prelude.graphql", BuiltIn: true}
	data := &codegen.Data{
		Config: &config.Config{
			Exec:    config.ExecConfig{Layout: config.ExecLayoutFollowSchema, DirName: dir},
			Sources: []*ast.Source{schema},
			Schema: &ast.Schema{Types: map[string]*ast.Definition{
				"Query":  {Name: "Query", Position: &ast.Position{Src: schema}},
				"String": {Name: "String", Position: &ast.Position{Src: prelude}},
			}},
		},
		ReferencedTypes: map[string]*config.TypeReference{"String": str},
	}
	calls, err := collectPluginCalls(data)
	require.NoError(t, err)
	require.Empty(t, calls)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "federation.go"), []byte(call), 0o600))
	calls, err = collectPluginCalls(data)
	require.NoError(t, err)
	require.Equal(t, map[string]callForms{str.UnmarshalFunc(): {method: true}}, calls)
}

// A package of the executor that does not exist yet has no calls.
func TestCollectPluginCallsWithoutPackage(t *testing.T) {
	data := &codegen.Data{Config: &config.Config{Exec: config.ExecConfig{
		Layout:   config.ExecLayoutSingleFile,
		Filename: filepath.Join(t.TempDir(), "graph", "generated.go"),
	}}}
	calls, err := collectPluginCalls(data)
	require.NoError(t, err)
	require.Empty(t, calls)
}

//go:generate rm -f resolver.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package omittypedinputunmarshalers

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

// omit_typed_input_unmarshalers suppresses the generated Unmarshal<Input> helpers, for
// schemas whose exec package would otherwise gain one exported name per input object.
// Absence cannot be asserted by referring to them, so read the generated package back.
func TestTypedUnmarshalersAreOmitted(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	fset := token.NewFileSet()
	var found []string
	for _, entry := range entries {
		name := entry.Name()
		if filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		require.NoErrorf(t, err, "parsing %s", name)

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Unmarshal") {
				found = append(found, fn.Name.Name)
			}
		}
	}

	assert.Empty(t, found, "the option must suppress every typed unmarshaler")
}

// Suppressing the helpers leaves the name-keyed API as the supported entry point rather
// than removing the capability.
func TestUnmarshalNamedInputFromContextStillWorks(t *testing.T) {
	var (
		called bool
		got    SearchFilters
		gotErr error
	)

	resolvers := &Stub{}
	resolvers.QueryResolver.Search = func(ctx context.Context, filters SearchFilters) (string, error) {
		called = true
		gotErr = graphql.UnmarshalNamedInputFromContext(
			ctx, "SearchFilters", map[string]any{"query": "hi"}, &got,
		)
		return "ok", nil
	}

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers}))

	var resp struct{ Search string }
	require.NoError(t, client.New(srv).Post(
		`query { search(filters: {query: "x"}) }`, &resp,
	))

	require.True(t, called, "the resolver never ran, so nothing was exercised")
	require.NoError(t, gotErr)
	require.NotNil(t, got.Query)
	assert.Equal(t, "hi", *got.Query)
}

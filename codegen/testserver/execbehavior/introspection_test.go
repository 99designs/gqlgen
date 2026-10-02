package execbehavior

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"

	"github.com/99designs/gqlgen/codegen/testserver/execbehavior/internal/exectest"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/introspection"
)

// TestIntrospectionSnapshot compares the responses to the full introspection query, and to
// a query for what it leaves out, with testdata/introspection.json. Run it with
// UPDATE_SNAPSHOT=1 to rewrite the file after changing introspection.graphql on purpose.
//
// The snapshot keeps the part of the schema that introspection.graphql and the prelude
// declare, so that the schemas of the other tests do not change it.
func TestIntrospectionSnapshot(t *testing.T) {
	srv := newServer(&Stub{}, DirectiveRoot{})
	srv.Use(extension.Introspection{})

	query := func(q string) map[string]any {
		w := exectest.RawPost(t, srv, q, nil)
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.NotContains(t, resp, "errors")
		return resp
	}

	full := query(introspection.Query)
	keepDeclared(t, full["data"].(map[string]any)["__schema"].(map[string]any))
	// The standard query leaves out isOneOf and deprecated input fields.
	details := query(`{
		oneOf: __type(name: "OneOfInput") { isOneOf inputFields { name } }
		input: __type(name: "DescribedInput") {
			isOneOf
			inputFields(includeDeprecated: true) { name description defaultValue isDeprecated deprecationReason }
		}
	}`)

	got, err := json.MarshalIndent(map[string]any{"full": full, "details": details}, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	const path = "testdata/introspection.json"
	if os.Getenv("UPDATE_SNAPSHOT") != "" {
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		require.NoError(t, os.WriteFile(path, got, 0o644))
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err)
	// Git may check the snapshot out with CRLF line endings on Windows.
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
	require.Equal(t, string(want), string(got))
}

// keepDeclared removes from schema, the __schema of an introspection response, the
// types, directives and Query fields that neither introspection.graphql nor the prelude
// declares.
func keepDeclared(t *testing.T, schema map[string]any) {
	t.Helper()

	src, err := os.ReadFile("introspection.graphql")
	require.NoError(t, err)
	var types, directives, queryFields []string
	for _, s := range []*ast.Source{validator.Prelude, {Name: "introspection.graphql", Input: string(src)}} {
		doc, err := parser.ParseSchema(s)
		require.NoError(t, err)
		for _, def := range doc.Definitions {
			types = append(types, def.Name)
		}
		for _, dir := range doc.Directives {
			directives = append(directives, dir.Name)
		}
		for _, ext := range doc.Extensions {
			if ext.Name == "Query" {
				for _, f := range ext.Fields {
					queryFields = append(queryFields, f.Name)
				}
			}
		}
	}

	keep := func(list any, names []string) []any {
		var res []any
		for _, v := range list.([]any) {
			if slices.Contains(names, v.(map[string]any)["name"].(string)) {
				res = append(res, v)
			}
		}
		return res
	}
	schema["directives"] = keep(schema["directives"], directives)
	schema["types"] = keep(schema["types"], append(types, "Query"))
	for _, typ := range schema["types"].([]any) {
		if typ := typ.(map[string]any); typ["name"] == "Query" {
			typ["fields"] = keep(typ["fields"], queryFields)
		}
	}
}

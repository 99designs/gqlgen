//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_legacy.yml -stub generated-legacy/stub.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_strict.yml -stub generated-strict/stub.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen_functions.yml -stub generated-functions/stub.go

package enumoutputs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	functions "github.com/99designs/gqlgen/codegen/testserver/enumoutputs/generated-functions"
	legacy "github.com/99designs/gqlgen/codegen/testserver/enumoutputs/generated-legacy"
	strict "github.com/99designs/gqlgen/codegen/testserver/enumoutputs/generated-strict"
	"github.com/99designs/gqlgen/codegen/testserver/enumoutputs/model"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

func TestEnumOutputs(t *testing.T) {
	bad := model.Status("BOGUS")
	empty := model.Status("")
	ok := model.StatusOk
	customBad := model.Custom("BOGUS")
	contextBad := model.Contextual("BOGUS")
	contextError := model.Contextual("ERROR")
	boundBad := model.Bound(99)
	boundOK := model.One
	customOK := model.Custom("OK")
	contextOK := model.Contextual("OK")
	for _, mode := range []string{"strict", "functions"} {
		t.Run(mode, func(t *testing.T) {
			for _, tc := range []struct {
				name    string
				query   string
				record  *model.Record
				data    string
				path    []any
				message string
			}{
				{
					name:   "valid",
					query:  `{ record { nullable required } }`,
					record: &model.Record{Nullable: &ok, Required: ok},
					data:   `{"record":{"nullable":"OK","required":"OK"}}`,
				},
				{
					name:   "valid custom marshalers",
					query:  `{ record { custom contextual } }`,
					record: &model.Record{Custom: &customOK, Contextual: &contextOK},
					data:   `{"record":{"custom":"OK","contextual":"OK"}}`,
				},
				{
					name:   "legitimate null",
					query:  `{ record { nullable bound custom contextual } }`,
					record: &model.Record{},
					data:   `{"record":{"nullable":null,"bound":null,"custom":null,"contextual":null}}`,
				},
				{
					name:   "invalid nullable",
					query:  `{ record { nullable __typename } }`,
					record: &model.Record{Nullable: &bad},
					data:   `{"record":{"nullable":null,"__typename":"Record"}}`,
					path:   []any{"record", "nullable"},
				},
				{
					name:   "aliased field",
					query:  `{ other: record { aliased: nullable } }`,
					record: &model.Record{Nullable: &bad},
					data:   `{"other":{"aliased":null}}`,
					path:   []any{"other", "aliased"},
				},
				{
					name:   "zero value",
					query:  `{ record { nullable } }`,
					record: &model.Record{Nullable: &empty},
					data:   `{"record":{"nullable":null}}`,
					path:   []any{"record", "nullable"},
				},
				{
					name:   "non-null field",
					query:  `{ record { required } }`,
					record: &model.Record{Required: bad},
					data:   `{"record":null}`,
					path:   []any{"record", "required"},
				},
				{
					name:   "non-null root",
					query:  `{ requiredRecord { required } }`,
					record: &model.Record{Required: bad},
					data:   `null`,
					path:   []any{"requiredRecord", "required"},
				},
				{
					name:   "nullable list item",
					query:  `{ record { items } }`,
					record: &model.Record{Items: []*model.Status{&ok, &bad, nil}},
					data:   `{"record":{"items":["OK",null,null]}}`,
					path:   []any{"record", "items", 1},
				},
				{
					name:   "required list item",
					query:  `{ record { requiredItems } }`,
					record: &model.Record{RequiredItems: []model.Status{ok, bad}},
					data:   `{"record":{"requiredItems":null}}`,
					path:   []any{"record", "requiredItems", 1},
				},
				{
					name:   "required list",
					query:  `{ record { requiredList } }`,
					record: &model.Record{RequiredList: []model.Status{ok, bad}},
					data:   `{"record":null}`,
					path:   []any{"record", "requiredList", 1},
				},
				{
					name:   "nested lists",
					query:  `{ record { nested } }`,
					record: &model.Record{Nested: [][]model.Status{{ok}, {ok, bad}}},
					data:   `{"record":{"nested":null}}`,
					path:   []any{"record", "nested", 1, 1},
				},
				{
					name:   "mapped valid",
					query:  `{ record { bound } }`,
					record: &model.Record{Bound: &boundOK},
					data:   `{"record":{"bound":"ONE"}}`,
				},
				{
					name:    "mapped invalid",
					query:   `{ record { bound } }`,
					record:  &model.Record{Bound: &boundBad},
					data:    `{"record":{"bound":null}}`,
					path:    []any{"record", "bound"},
					message: "invalid value for enum Bound",
				},
				{
					name:    "custom invalid",
					query:   `{ record { custom } }`,
					record:  &model.Record{Custom: &customBad},
					data:    `{"record":{"custom":null}}`,
					path:    []any{"record", "custom"},
					message: "invalid value for enum Custom",
				},
				{
					name:    "context invalid",
					query:   `{ record { contextual } }`,
					record:  &model.Record{Contextual: &contextBad},
					data:    `{"record":{"contextual":null}}`,
					path:    []any{"record", "contextual"},
					message: "invalid value for enum Contextual",
				},
				{
					name:    "context required error",
					query:   `{ record { requiredContextual } }`,
					record:  &model.Record{RequiredContextual: contextError},
					data:    `{"record":null}`,
					path:    []any{"record", "requiredContextual"},
					message: "context enum failed",
				},
				{
					name:    "context list error",
					query:   `{ record { contextualItems } }`,
					record:  &model.Record{ContextualItems: []model.Contextual{"OK", contextError}},
					data:    `{"record":{"contextualItems":null}}`,
					path:    []any{"record", "contextualItems", 1},
					message: "context enum failed",
				},
				{
					name:    "context error",
					query:   `{ record { contextual } }`,
					record:  &model.Record{Contextual: &contextError},
					data:    `{"record":{"contextual":null}}`,
					path:    []any{"record", "contextual"},
					message: "context enum failed",
				},
			} {
				t.Run(tc.name, func(t *testing.T) {
					response := execute(t, mode, tc.record, tc.query)
					require.JSONEq(t, tc.data, string(response.Data))
					if tc.path == nil {
						require.Empty(t, response.Errors)
						return
					}
					require.Len(t, response.Errors, 1)
					expectedPath, err := json.Marshal(tc.path)
					require.NoError(t, err)
					require.JSONEq(t, string(expectedPath), string(response.Errors[0].Path))
					message := tc.message
					if message == "" {
						message = "invalid value for enum Status"
					}
					require.Equal(t, message, response.Errors[0].Message)
				})
			}
		})
	}
}

func TestLegacyEnumOutputs(t *testing.T) {
	bad := model.Status("BOGUS")
	response := execute(t, "legacy", &model.Record{Nullable: &bad}, `{ record { nullable } }`)
	require.Empty(t, response.Errors)
	require.JSONEq(t, `{"record":{"nullable":"BOGUS"}}`, string(response.Data))
}

type response struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string          `json:"message"`
		Path    json.RawMessage `json:"path"`
	} `json:"errors"`
}

func execute(t *testing.T, mode string, record *model.Record, query string) response {
	t.Helper()
	resolve := func(context.Context) (*model.Record, error) { return record, nil }
	var schema graphql.ExecutableSchema
	switch mode {
	case "strict":
		resolvers := &strict.Stub{}
		resolvers.QueryResolver.Record = resolve
		resolvers.QueryResolver.RequiredRecord = resolve
		schema = strict.NewExecutableSchema(strict.Config{Resolvers: resolvers})
	case "functions":
		resolvers := &functions.Stub{}
		resolvers.QueryResolver.Record = resolve
		resolvers.QueryResolver.RequiredRecord = resolve
		schema = functions.NewExecutableSchema(functions.Config{Resolvers: resolvers})
	case "legacy":
		resolvers := &legacy.Stub{}
		resolvers.QueryResolver.Record = resolve
		resolvers.QueryResolver.RequiredRecord = resolve
		schema = legacy.NewExecutableSchema(legacy.Config{Resolvers: resolvers})
	default:
		t.Fatalf("unknown fixture mode %q", mode)
	}
	body, err := json.Marshal(map[string]string{"query": query})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.NewDefaultServer(schema).ServeHTTP(recorder, request)
	require.Equal(t, 200, recorder.Code, recorder.Body.String())
	var result response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result), recorder.Body.String())
	return result
}

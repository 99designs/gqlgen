package apollotracing_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/apollotracing"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

// todo: extract out common code for testing handler plugins without requiring a codegenned server.
func TestApolloTracing(t *testing.T) {
	now := time.Unix(0, 0)

	graphql.Now = func() time.Time {
		defer func() {
			now = now.Add(100 * time.Nanosecond)
		}()
		return now
	}

	schema := gqlparser.MustLoadSchema(&ast.Source{Input: `
		schema { query: Query }
		type Query {
			me: User!
			user(id: Int): User!
		}
		type User { name: String! }
	`})

	es := &graphql.ExecutableSchemaMock{
		QueryFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			// Field execution happens inside the generated code, we want just enough to test against right now.
			ctx = graphql.WithResolverContext(ctx, &graphql.ResolverContext{
				Object: "Query",
				Field: graphql.CollectedField{
					Field: &ast.Field{
						Name:       "me",
						Alias:      "me",
						Definition: schema.Types["Query"].Fields.ForName("me"),
					},
				},
			})
			res, err := graphql.GetRequestContext(ctx).ResolverMiddleware(ctx, func(ctx context.Context) (interface{}, error) {
				return &graphql.Response{Data: []byte(`{"name":"test"}`)}, nil
			})
			require.NoError(t, err)
			return res.(*graphql.Response)
		},
		MutationFunc: func(ctx context.Context, op *ast.OperationDefinition) *graphql.Response {
			return graphql.ErrorResponse(ctx, "mutations are not supported")
		},
		SchemaFunc: func() *ast.Schema {
			return schema
		},
	}
	h := handler.New(es)
	h.AddTransport(transport.POST{})
	h.Use(apollotracing.New())

	resp := doRequest(h, "POST", "/graphql", `{"query":"{ me { name } }"}`)
	assert.Equal(t, http.StatusOK, resp.Code)
	var respData struct {
		Extensions struct {
			Tracing apollotracing.TracingExtension `json:"tracing"`
		} `json:"extensions"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &respData))

	tracing := respData.Extensions.Tracing

	require.EqualValues(t, 1, tracing.Version)

	require.EqualValues(t, 0, tracing.StartTime.UnixNano())
	require.EqualValues(t, 700, tracing.EndTime.UnixNano())
	require.EqualValues(t, 700, tracing.Duration)

	require.EqualValues(t, 100, tracing.Parsing.StartOffset)
	require.EqualValues(t, 100, tracing.Parsing.Duration)

	require.EqualValues(t, 300, tracing.Validation.StartOffset)
	require.EqualValues(t, 100, tracing.Validation.Duration)

	require.EqualValues(t, 500, tracing.Execution.Resolvers[0].StartOffset)
	require.EqualValues(t, 100, tracing.Execution.Resolvers[0].Duration)
	require.EqualValues(t, []interface{}{"me"}, tracing.Execution.Resolvers[0].Path)
	require.EqualValues(t, "Query", tracing.Execution.Resolvers[0].ParentType)
	require.EqualValues(t, "me", tracing.Execution.Resolvers[0].FieldName)
	require.EqualValues(t, "User!", tracing.Execution.Resolvers[0].ReturnType)

}

func doRequest(handler http.Handler, method string, target string, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w
}

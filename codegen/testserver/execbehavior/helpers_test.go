//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package execbehavior

import (
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/testserver/execbehavior/internal/exectest"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// newServer returns a server for the schema with the given resolvers and directives.
func newServer(resolvers *Stub, directives DirectiveRoot) *handler.Server {
	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers, Directives: directives}))
	srv.AddTransport(transport.POST{})
	return srv
}

// post sends query to srv with exectest.Post.
func post(t *testing.T, srv http.Handler, query string) string {
	t.Helper()
	return exectest.Post(t, srv, query, nil)
}

// postVars is post with variables.
func postVars(t *testing.T, srv http.Handler, query string, variables map[string]any) string {
	t.Helper()
	return exectest.Post(t, srv, query, variables)
}

// execute runs query on es and returns every response, including the deferred ones
// that follow the first, as a JSON array. The deferred responses are sorted by label
// and path, and the errors of each response are normalized by exectest.NormalizeErrors,
// so that the result does not depend on the order in which concurrent fields finished.
func execute(t *testing.T, es graphql.ExecutableSchema, query string) string {
	t.Helper()

	exec := executor.New(es)
	ctx := graphql.StartOperationTrace(context.Background())
	opCtx, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{Query: query})
	require.Empty(t, errs)
	handler, ctx := exec.DispatchOperation(ctx, opCtx)

	var responses []*graphql.Response
	for range 100 {
		resp := handler(ctx)
		if resp == nil {
			break
		}
		exectest.NormalizeErrors(resp.Errors)
		responses = append(responses, resp)
	}
	if len(responses) > 1 {
		rest := responses[1:]
		slices.SortStableFunc(rest, func(a, b *graphql.Response) int {
			return cmp.Or(cmp.Compare(a.Label, b.Label), exectest.ComparePaths(a.Path, b.Path))
		})
		// Which deferred response comes last, and so says that nothing is left, depends
		// on timing; the first response always says that more follow.
		for _, resp := range rest {
			resp.HasNext = nil
		}
	}
	out, err := json.Marshal(responses)
	require.NoError(t, err)
	return string(out)
}

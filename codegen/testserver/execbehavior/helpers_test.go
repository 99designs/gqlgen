//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package execbehavior

import (
	"net/http"
	"testing"

	"github.com/99designs/gqlgen/codegen/testserver/execbehavior/internal/exectest"
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

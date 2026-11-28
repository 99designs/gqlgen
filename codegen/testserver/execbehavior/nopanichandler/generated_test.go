//go:generate go run ../../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package nopanichandler

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/testserver/execbehavior/internal/exectest"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestOmitPanicHandler(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Ping = func(ctx context.Context) (string, error) {
		return "pong", nil
	}
	resolvers.QueryResolver.Panicking = func(ctx context.Context) (*string, error) {
		panic("resolver panicked")
	}
	resolvers.QueryResolver.PanickingObject = func(ctx context.Context) (*PanicObject, error) {
		return &PanicObject{Name: "object"}, nil
	}
	resolvers.QueryResolver.PanickingList = func(ctx context.Context) ([]*PanicObject, error) {
		return []*PanicObject{{Name: "element"}}, nil
	}
	resolvers.MutationResolver.Panicking = func(ctx context.Context) (*string, error) {
		panic("mutation panicked")
	}
	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})

	post := func(query string) (int, string) {
		w := exectest.RawPost(t, srv, query, nil)
		return w.Code, w.Body.String()
	}

	code, body := post(`{ ping }`)
	require.Equal(t, http.StatusOK, code)
	require.JSONEq(t, `{"data":{"ping":"pong"}}`, body)

	// Only panics in fields resolved on the request's goroutine are tested: without a
	// panic handler, a panic in a field resolved concurrently would end the process.
	failed := `{"errors":[{"message":"internal system error"}],"data":null}`
	for _, query := range []string{
		`{ panicking }`,
		`{ panickingObject { name boom } }`,
		`{ panickingList { boom } }`,
		`mutation { panicking }`,
	} {
		code, body := post(query)
		require.Equal(t, http.StatusInternalServerError, code, query)
		require.JSONEq(t, failed, body, query)
	}
}

package testserver

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
)

func TestPanics(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Panics = func(ctx context.Context) (panics *Panics, e error) {
		return &Panics{}, nil
	}
	resolvers.PanicsResolver.ArgUnmarshal = func(ctx context.Context, obj *Panics, u []MarshalPanic) (b bool, e error) {
		return true, nil
	}
	resolvers.PanicsResolver.FieldScalarMarshal = func(ctx context.Context, obj *Panics) (marshalPanic []MarshalPanic, e error) {
		return []MarshalPanic{MarshalPanic("aa"), MarshalPanic("bb")}, nil
	}

	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolvers})))
	c := client.New(srv.URL)

	t.Run("panics in marshallers will not kill server", func(t *testing.T) {
		var resp interface{}
		err := c.Post(`query { panics { fieldScalarMarshal } }`, &resp)

		require.EqualError(t, err, "http 422: {\"errors\":[{\"message\":\"internal system error\"}],\"data\":null}")
	})

	t.Run("panics in unmarshalers will not kill server", func(t *testing.T) {
		var resp interface{}
		err := c.Post(`query { panics { argUnmarshal(u: ["aa", "bb"]) } }`, &resp)

		require.EqualError(t, err, "http 422: {\"errors\":[{\"message\":\"internal system error\"}],\"data\":null}")
	})

	t.Run("panics in funcs unmarshal return errors", func(t *testing.T) {
		var resp interface{}
		err := c.Post(`query { panics { fieldFuncMarshal(u: ["aa", "bb"]) } }`, &resp)

		require.EqualError(t, err, "http 422: {\"errors\":[{\"message\":\"internal system error\"}],\"data\":null}")
	})

	t.Run("panics in funcs marshal return errors", func(t *testing.T) {
		var resp interface{}
		err := c.Post(`query { panics { fieldFuncMarshal(u: []) } }`, &resp)

		require.EqualError(t, err, "http 422: {\"errors\":[{\"message\":\"internal system error\"}],\"data\":null}")
	})
}

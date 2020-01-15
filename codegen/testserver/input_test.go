package testserver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
)

func TestInput(t *testing.T) {
	resolvers := &Stub{}
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers}))
	c := client.New(srv)

	t.Run("when function errors on directives", func(t *testing.T) {
		resolvers.QueryResolver.InputSlice = func(ctx context.Context, arg []string) (b bool, e error) {
			return true, nil
		}

		var resp struct {
			DirectiveArg *string
		}

		err := c.Post(`query { inputSlice(arg: ["ok", 1, 2, "ok"]) }`, &resp)

		require.EqualError(t, err, `http 422: {"errors":[{"message":"Expected type String!, found 1.","locations":[{"line":1,"column":32}],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}},{"message":"Expected type String!, found 2.","locations":[{"line":1,"column":35}],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`)
		require.Nil(t, resp.DirectiveArg)
	})
}

//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package followschema

import (
	"context"
	"reflect"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/codegen/testserver/nullabledirectives/generated"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

// isNil checks if the given value is nil
func isNil(input any) bool {
	if input == nil {
		return true
	}
	// Using reflect to check if the value is nil. This is necessary for
	// any types that are not nil types but have a nil value (e.g. *string).
	value := reflect.ValueOf(input)
	return value.IsNil()
}

func TestDirectives(t *testing.T) {
	resolvers := &Stub{}
	ok := "Ok"
	resolvers.QueryResolver.DirectiveSingleNullableArg = func(ctx context.Context, arg1 *string) (*string, error) {
		if arg1 != nil {
			return arg1, nil
		}

		return &ok, nil
	}

	srv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolvers,
		Directives: generated.DirectiveRoot{
			Populate: func(ctx context.Context, obj any, next graphql.Resolver, value string) (any, error) {
				res, err := next(ctx)
				if err != nil {
					return nil, err
				}

				if !isNil(res) {
					return res, err
				}

				return &value, nil
			},
			Noop: func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
				return next(ctx)
			},
		},
	}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("arg directives", func(t *testing.T) {
		t.Run("directive is called with null arguments", func(t *testing.T) {
			var resp struct {
				DirectiveSingleNullableArg *string
			}

			err := c.Post(`query { directiveSingleNullableArg }`, &resp)

			require.NoError(t, err)
			require.Equal(t, "test", *resp.DirectiveSingleNullableArg)
		})
	})
}

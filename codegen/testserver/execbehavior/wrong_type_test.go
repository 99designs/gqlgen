package execbehavior

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

func TestDirectiveAndMiddlewareReturnAnotherType(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.WrongTypes = func(ctx context.Context) (*WrongTypes, error) {
		s := "s"
		return &WrongTypes{
			IntAsString:          &s,
			NonNullIntAsString:   s,
			Marshaler:            s,
			ReplacedByMiddleware: &s,
		}, nil
	}
	resolvers.QueryResolver.WrongTypeArg = func(ctx context.Context, value *string) (*string, error) {
		return value, nil
	}
	resolvers.QueryResolver.WrongTypeInput = func(ctx context.Context, input WrongTypeInput) (*string, error) {
		return input.Value, nil
	}
	srv := newServer(resolvers, DirectiveRoot{
		ReturnValue: func(ctx context.Context, obj any, next graphql.Resolver, kind string) (any, error) {
			if _, err := next(ctx); err != nil {
				return nil, err
			}
			if kind == "marshaler" {
				return graphql.MarshalString("from the directive"), nil
			}
			return 42, nil
		},
	})
	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (any, error) {
		res, err := next(ctx)
		if graphql.GetFieldContext(ctx).Field.Name == "replacedByMiddleware" {
			return 7, err
		}
		return res, err
	})

	t.Run("nullable field", func(t *testing.T) {
		got := post(t, srv, `{ wrongTypes { intAsString } }`)
		require.JSONEq(t, `{"data":{"wrongTypes":{"intAsString":null}},"errors":[
			{"message":"unexpected type int from middleware/directive chain, should be *string","path":["wrongTypes","intAsString"]}]}`, got)
	})

	t.Run("non-null field", func(t *testing.T) {
		got := post(t, srv, `{ wrongTypes { nonNullIntAsString } }`)
		require.JSONEq(t, `{"data":{"wrongTypes":null},"errors":[
			{"message":"unexpected type int from middleware/directive chain, should be string","path":["wrongTypes","nonNullIntAsString"]}]}`, got)
	})

	t.Run("field resolved to a marshaler", func(t *testing.T) {
		got := post(t, srv, `{ wrongTypes { marshaler } }`)
		require.JSONEq(t, `{"data":{"wrongTypes":{"marshaler":"from the directive"}}}`, got)
	})

	t.Run("field replaced by middleware", func(t *testing.T) {
		got := post(t, srv, `{ wrongTypes { replacedByMiddleware } }`)
		require.JSONEq(t, `{"data":{"wrongTypes":{"replacedByMiddleware":null}},"errors":[
			{"message":"unexpected type int from middleware/directive chain, should be *string","path":["wrongTypes","replacedByMiddleware"]}]}`, got)
	})

	t.Run("argument", func(t *testing.T) {
		got := post(t, srv, `{ wrongTypeArg(value: "v") }`)
		require.JSONEq(t, `{"data":{"wrongTypeArg":null},"errors":[
			{"message":"unexpected type int from directive, should be *string","path":["wrongTypeArg","value"]}]}`, got)
	})

	t.Run("input field", func(t *testing.T) {
		got := post(t, srv, `{ wrongTypeInput(input: { value: "v" }) }`)
		require.JSONEq(t, `{"data":{"wrongTypeInput":null},"errors":[
			{"message":"unexpected type int from directive, should be *string","path":["wrongTypeInput","input","value"]}]}`, got)
	})
}

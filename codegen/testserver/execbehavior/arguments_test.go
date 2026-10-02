package execbehavior

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/testserver/execbehavior/internal/exectest"
	"github.com/99designs/gqlgen/graphql"
)

func TestArguments(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.ArgProbe = func(ctx context.Context, strict *Strict, withDefault *int, plain *string) (*string, error) {
		describe := func(v any) string {
			switch v := v.(type) {
			case *Strict:
				if v != nil {
					return string(*v)
				}
			case *int:
				if v != nil {
					return strconv.Itoa(*v)
				}
			case *string:
				if v != nil {
					return *v
				}
			}
			return "nil"
		}
		s := describe(strict) + "," + describe(withDefault) + "," + describe(plain)
		return &s, nil
	}
	resolvers.QueryResolver.ArgObject = func(ctx context.Context) (*ArgObject, error) {
		return &ArgObject{Ok: "ok"}, nil
	}
	resolvers.ArgObjectResolver.Echo = func(ctx context.Context, obj *ArgObject, strict Strict) (*string, error) {
		s := string(strict)
		return &s, nil
	}
	resolvers.QueryResolver.WrongTypeArg = func(ctx context.Context, value *string) (*string, error) {
		return value, nil
	}
	srv := newServer(resolvers, DirectiveRoot{})

	t.Run("omitted, default and explicit null", func(t *testing.T) {
		got := post(t, srv, `{
			omitted: argProbe
			given: argProbe(strict: "s", withDefault: 1, plain: "p")
			nulls: argProbe(strict: null, withDefault: null, plain: null)
		}`)
		require.JSONEq(
			t,
			`{"data":{"omitted":"nil,7,nil","given":"s,1,p","nulls":"nil,nil,nil"}}`,
			got,
		)
	})

	t.Run("invalid literal", func(t *testing.T) {
		got := post(t, srv, `{ argProbe(strict: "bad") }`)
		require.JSONEq(t, `{"data":{"argProbe":null},"errors":[
			{"message":"strict rejects this value","path":["argProbe","strict"]}]}`, got)
	})

	t.Run("invalid variable", func(t *testing.T) {
		// The raw response is compared, as post would remove the locations that are
		// missing here.
		w := exectest.RawPost(
			t,
			srv,
			`query($s: Strict) { argProbe(strict: $s) }`,
			map[string]any{"s": "bad"},
		)
		// KNOWN BUG: an invalid variable fails only the field that uses it, with an error
		// without locations.
		// Expected: a request error and no data, because the spec coerces the values of
		// the variables before it executes the operation, and fails the request when one
		// cannot be coerced.
		// Issue: https://github.com/99designs/gqlgen/issues/4460
		require.JSONEq(t, `{"data":{"argProbe":null},"errors":[
			{"message":"strict rejects this value","path":["argProbe","strict"]}]}`,
			w.Body.String())
	})

	t.Run("invalid argument of a nested field", func(t *testing.T) {
		got := post(t, srv, `{ argObject { ok echo(strict: "bad") good: echo(strict: "good") } }`)
		require.JSONEq(t, `{"data":{"argObject":{"ok":"ok","echo":null,"good":"good"}},"errors":[
			{"message":"strict rejects this value","path":["argObject","echo","strict"]}]}`, got)
	})

	// wrongTypeArg and @returnValue are from wrong_type.graphql.
	t.Run("directives of an omitted argument are not called", func(t *testing.T) {
		called := false
		srv := newServer(resolvers, DirectiveRoot{
			ReturnValue: func(ctx context.Context, obj any, next graphql.Resolver, kind string) (any, error) {
				called = true
				return next(ctx)
			},
		})
		got := post(t, srv, `{ wrongTypeArg }`)
		require.JSONEq(t, `{"data":{"wrongTypeArg":null}}`, got)
		require.False(t, called)
	})
}

func TestDirectiveArguments(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.StrictDirective = func(ctx context.Context) (*string, error) {
		return new("resolved"), nil
	}
	resolvers.QueryResolver.BadStrictDirective = func(ctx context.Context) (*string, error) {
		return new("resolved"), nil
	}
	srv := newServer(resolvers, DirectiveRoot{
		StrictArg: func(ctx context.Context, obj any, next graphql.Resolver, value Strict) (any, error) {
			res, err := next(ctx)
			if err != nil {
				return nil, err
			}
			return new(*res.(*string) + " with " + string(value)), nil
		},
	})

	require.JSONEq(
		t,
		`{"data":{"strictDirective":"resolved with good"}}`,
		post(t, srv, `{ strictDirective }`),
	)
	require.JSONEq(
		t,
		`{"data":{"badStrictDirective":null},"errors":[{"message":"strict rejects this value","path":["badStrictDirective"]}]}`,
		post(t, srv, `{ badStrictDirective }`),
	)
}

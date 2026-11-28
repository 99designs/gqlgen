package tableimpl

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
)

func TestChain(t *testing.T) {
	var log []string
	tag := &DirectiveDef[*testEC]{
		Name: "tag",
		Call: func(ctx context.Context, ec *testEC, obj any, next graphql.Resolver, args map[string]any) (any, error) {
			log = append(log, "before "+args["name"].(string))
			res, err := next(ctx)
			log = append(log, "after "+args["name"].(string))
			return fmt.Sprintf("%s(%v)", args["name"], res), err
		},
		args: []arg[*testEC]{
			{name: "name", typ: InOf(func(_ context.Context, _ *testEC, v any) (string, error) {
				if v == "bad" {
					return "", errors.New("bad name")
				}
				return fmt.Sprint(v), nil
			}).In},
		},
	}
	next := func(ctx context.Context) (any, error) { return "value", nil }

	res, err := chain(
		nil,
		nil,
		[]directive[*testEC]{
			{def: tag, args: map[string]any{"name": "inner"}},
			{def: tag, args: map[string]any{"name": "outer"}},
		},
		next,
		failure{},
	)(
		context.Background(),
	)
	require.NoError(t, err)
	require.Equal(t, "outer(inner(value))", res)
	require.Equal(t, []string{"before outer", "before inner", "after inner", "after outer"}, log)

	// A directive whose arguments fail returns the zero value of the element.
	res, err = chain(
		nil,
		nil,
		[]directive[*testEC]{{def: tag, args: map[string]any{"name": "bad"}}},
		next,
		failure{zero: (*int)(nil)},
	)(
		context.Background(),
	)
	require.EqualError(t, err, "bad name")
	require.Equal(t, (*int)(nil), res)

	res, err = chain[*testEC](nil, nil, nil, next, failure{})(context.Background())
	require.NoError(t, err)
	require.Equal(t, "value", res)
}

func TestChainNotImplemented(t *testing.T) {
	missing := &DirectiveDef[*testEC]{
		Name: "missing",
		Call: func(context.Context, *testEC, any, graphql.Resolver, map[string]any) (any, error) {
			t.Fatal("the directive without an implementation is called")
			return nil, nil
		},
		implemented: func(*testEC) bool { return false },
	}
	next := func(ctx context.Context) (any, error) { return "value", nil }
	dirs := []directive[*testEC]{{def: missing}}

	res, err := chain(nil, nil, dirs, next, failure{zero: (*int)(nil)})(context.Background())
	require.EqualError(t, err, "directive missing is not implemented")
	require.Equal(t, (*int)(nil), res)

	// The INPUT_OBJECT directives put the error on the path, as the functions mode does.
	ctx := graphql.WithPathContext(context.Background(), graphql.NewPathWithField("input"))
	res, err = chain(nil, nil, dirs, next, failure{zero: "it", onPath: true})(ctx)
	var gqlErr *gqlerror.Error
	require.ErrorAs(t, err, &gqlErr)
	require.Equal(t, ast.Path{ast.PathName("input")}, gqlErr.Path)
	require.Equal(t, "it", res)

	// A directive with an implementation passes on the errors of what it wraps, whatever
	// they say.
	pass := &DirectiveDef[*testEC]{
		Name: "pass",
		Call: func(ctx context.Context, _ *testEC, _ any, next graphql.Resolver, _ map[string]any) (any, error) {
			return next(ctx)
		},
		implemented: func(*testEC) bool { return true },
	}
	failing := func(context.Context) (any, error) {
		return nil, errors.New("directive pass is not implemented yet")
	}
	_, err = chain(
		nil,
		nil,
		[]directive[*testEC]{{def: pass}},
		failing,
		failure{zero: 0},
	)(
		context.Background(),
	)
	require.EqualError(t, err, "directive pass is not implemented yet")
}

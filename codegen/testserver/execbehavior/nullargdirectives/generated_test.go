//go:generate go run ../../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package nullargdirectives

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/testserver/execbehavior/internal/exectest"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestArgumentDirectivesWithNull(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Echo = func(ctx context.Context, arg *string) (*string, error) {
		return arg, nil
	}
	// received describes what the directive received from the argument.
	var received []string
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			Fill: func(ctx context.Context, obj any, next graphql.Resolver, value string) (any, error) {
				res, err := next(ctx)
				if err != nil {
					return nil, err
				}
				if s, ok := res.(*string); ok && s != nil {
					received = append(received, *s)
					return s, nil
				}
				received = append(received, fmt.Sprintf("%T(nil)", res))
				return &value, nil
			},
		},
	}))
	srv.AddTransport(transport.POST{})

	for _, tc := range []struct{ query, want string }{
		{`{ echo(arg: "given") }`, `{"data":{"echo":"given"}}`},
		{`{ echo }`, `{"data":{"echo":"filled"}}`},
		{`{ echo(arg: null) }`, `{"data":{"echo":"filled"}}`},
	} {
		require.JSONEq(t, tc.want, exectest.Post(t, srv, tc.query, nil), tc.query)
	}
	require.Equal(t, []string{"given", "*string(nil)", "*string(nil)"}, received)
}

func TestDirectiveArgumentOfInterfaceType(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Tagged = func(ctx context.Context) (*string, error) {
		s := "tagged"
		return &s, nil
	}
	resolvers.QueryResolver.TaggedWith = func(ctx context.Context) (*string, error) {
		s := "taggedWith"
		return &s, nil
	}
	// received holds, by field, what the directive received as its argument.
	received := map[string]string{}
	var mu sync.Mutex
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			Tag: func(ctx context.Context, obj any, next graphql.Resolver, tag Tag) (any, error) {
				got := "nil"
				if tag != nil {
					got = tag.Tag()
				}
				mu.Lock()
				received[graphql.GetFieldContext(ctx).Field.Name] = got
				mu.Unlock()
				return next(ctx)
			},
		},
	}))
	srv.AddTransport(transport.POST{})

	require.JSONEq(
		t,
		`{"data":{"tagged":"tagged","taggedWith":"taggedWith"}}`,
		exectest.Post(t, srv, `{ tagged taggedWith }`, nil),
	)
	// The absent argument is nil, not a panic on a type assertion of nil.
	require.Equal(t, map[string]string{"tagged": "nil", "taggedWith": "given"}, received)
}

//go:generate go run ../../testdata/gqlgen.go

package contextpropagation

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type ctxKey string

func getTodoContext(ctx context.Context) *string {
	if value, ok := ctx.Value(ctxKey("todoContext")).(string); ok {
		return &value
	}
	return nil
}

func New() Config {
	c := Config{
		Resolvers: new(Resolver),
	}
	c.Directives.WithContext = func(ctx context.Context, obj any, next graphql.Resolver, value string) (any, error) {
		return next(context.WithValue(ctx, ctxKey("todoContext"), value))
	}
	return c
}

type Resolver struct{}

func (r *Resolver) Query() QueryResolver {
	return r
}

func (r *Resolver) TodoContext() TodoContextResolver {
	return r
}

func (r *Resolver) TestTodo(ctx context.Context) (*Todo, error) {
	return &Todo{Text: "Test", Context: new(TodoContext)}, nil
}

func (r *Resolver) Value(ctx context.Context, obj *TodoContext) (*string, error) {
	return getTodoContext(ctx), nil
}

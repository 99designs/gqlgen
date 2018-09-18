package testserver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

func CustomDirective(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
	return next(ctx)
}

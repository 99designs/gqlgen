package testserver

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

func CustomDirective(ctx context.Context, obj interface{}, next graphql.Resolver, arg *ComplexInput) (res interface{}, err error) {
	return "CustomDirective", nil
}

package middleware

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

// ErrorPresenter transforms errors found while resolving into errors that will be returned to the user. It provides
// a good place to add any extra fields, like error.type, that might be desired by your frontend. Check the default
// implementation in graphql.DefaultErrorPresenter for an example.
func ErrorPresenter(ep graphql.ErrorPresenterFunc) graphql.Middleware {
	return func(next graphql.Handler) graphql.Handler {
		return func(ctx context.Context, writer graphql.Writer) {
			graphql.GetRequestContext(ctx).ErrorPresenter = ep
			next(ctx, writer)
		}
	}
}

// RecoverFunc is called to recover from panics inside goroutines. It can be used to send errors to error trackers
// and hide internal error types from clients.
func RecoverFunc(recover graphql.RecoverFunc) graphql.Middleware {
	return func(next graphql.Handler) graphql.Handler {
		return func(ctx context.Context, writer graphql.Writer) {
			graphql.GetRequestContext(ctx).Recover = recover
			next(ctx, writer)
		}
	}
}

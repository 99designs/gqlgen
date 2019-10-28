package handler

import (
	"context"

	"github.com/99designs/gqlgen/graphql/handler/transport"

	"github.com/99designs/gqlgen/graphql"
)

// ErrorPresenter transforms errors found while resolving into errors that will be returned to the user. It provides
// a good place to add any extra fields, like error.type, that might be desired by your frontend. Check the default
// implementation in graphql.DefaultErrorPresenter for an example.
func ErrorPresenter(ep graphql.ErrorPresenterFunc) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, writer transport.Writer) {
			graphql.GetRequestContext(ctx).ErrorPresenter = ep
			next(ctx, writer)
		}
	}
}

// RecoverFunc is called to recover from panics inside goroutines. It can be used to send errors to error trackers
// and hide internal error types from clients.
func RecoverFunc(recover graphql.RecoverFunc) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, writer transport.Writer) {
			graphql.GetRequestContext(ctx).Recover = recover
			next(ctx, writer)
		}
	}
}

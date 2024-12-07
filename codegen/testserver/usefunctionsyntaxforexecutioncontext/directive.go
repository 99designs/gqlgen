package usefunctionsyntaxforexecutioncontext

import (
	"context"
	"log"

	"github.com/99designs/gqlgen/graphql"
)

// LogDirective implementation
func LogDirective(ctx context.Context, obj any, next graphql.Resolver, message *string) (res any, err error) {
	log.Printf("Log Directive: %s", *message)

	// Proceed with the next resolver in the chain
	return next(ctx)
}

package fedruntime

import (
	"context"
	"fmt"
)

// ResolverFunc is a function that resolves a value in the context of federation entity resolution.
// It matches the signature used throughout the GraphQL execution pipeline.
type ResolverFunc func(context.Context) (any, error)

// DirectiveFunc wraps a resolver with directive middleware logic.
// It receives the context and the next resolver in the chain, and returns the result.
type DirectiveFunc func(context.Context, ResolverFunc) (any, error)

// ChainDirectives applies a chain of directives to a base resolver function.
// Directives are applied in reverse order, with each directive wrapping the next one.
func ChainDirectives(
	ctx context.Context, base ResolverFunc, directives []DirectiveFunc,
) (any, error) {
	if len(directives) == 0 {
		return base(ctx)
	}

	// Build chain from the end to the beginning (outermost to innermost)
	// The last directive in the slice wraps the base resolver
	// Each previous directive wraps the result of the next directive
	resolver := base
	for i := len(directives) - 1; i >= 0; i-- {
		directive := directives[i]
		next := resolver
		resolver = func(ctx context.Context) (any, error) {
			return directive(ctx, next)
		}
	}

	return resolver(ctx)
}

// WrapEntityResolver wraps an entity resolver with directive middleware.
// If no directives are provided, the resolver is called directly.
// Otherwise, directives are applied and the result is type-checked.
func WrapEntityResolver[T any](
	ctx context.Context,
	typedResolver func(context.Context) (T, error),
	directives []DirectiveFunc,
) (T, error) {
	var zero T

	// Fast path: no directives, call resolver directly
	if len(directives) == 0 {
		return typedResolver(ctx)
	}

	// Slow path: wrap with directives
	// Convert typed resolver to untyped for directive chain
	base := func(ctx context.Context) (any, error) {
		return typedResolver(ctx)
	}

	result, err := ChainDirectives(ctx, base, directives)
	if err != nil {
		return zero, err
	}

	// Type assert the result back to the expected type
	typedResult, ok := result.(T)
	if !ok {
		return zero, fmt.Errorf(
			"unexpected type %T from directive chain, expected %T",
			result,
			zero,
		)
	}

	return typedResult, nil
}

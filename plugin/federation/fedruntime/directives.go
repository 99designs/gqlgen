package fedruntime

import "context"

// ResolverFunc is a function that resolves a value in the context of federation entity resolution.
// It matches the signature used throughout the GraphQL execution pipeline.
type ResolverFunc func(context.Context) (any, error)

// DirectiveFunc wraps a resolver with directive middleware logic.
// It receives the context and the next resolver in the chain, and returns the result.
type DirectiveFunc func(context.Context, ResolverFunc) (any, error)

// ChainDirectives applies a chain of directives to a base resolver function.
// Directives are applied in order, with each directive wrapping the next one.
// This is the core directive chaining logic, extracted from code generation templates
// to improve testability and maintainability.
//
// Example:
//
//	base := func(ctx context.Context) (any, error) {
//	    return resolveEntity(ctx, id)
//	}
//	directives := []DirectiveFunc{
//	    func(ctx context.Context, next ResolverFunc) (any, error) {
//	        // auth directive logic
//	        return authMiddleware(ctx, next)
//	    },
//	    func(ctx context.Context, next ResolverFunc) (any, error) {
//	        // logging directive logic
//	        return loggingMiddleware(ctx, next)
//	    },
//	}
//	result, err := ChainDirectives(ctx, base, directives)
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

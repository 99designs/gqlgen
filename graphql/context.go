package graphql

import (
	"context"

	"github.com/vektah/gqlgen/neelance/errors"
	"github.com/vektah/gqlgen/neelance/query"
)

type Resolver func(ctx context.Context) (res interface{}, err error)
type ResolverMiddleware func(ctx context.Context, next Resolver) (res interface{}, err error)

type RequestContext struct {
	errors.Builder

	Variables  map[string]interface{}
	Doc        *query.Document
	Recover    RecoverFunc
	Middleware ResolverMiddleware
}

type key string

const (
	request  key = "request_context"
	resolver key = "resolver_context"
)

func GetRequestContext(ctx context.Context) *RequestContext {
	val := ctx.Value(request)
	if val == nil {
		return nil
	}

	return val.(*RequestContext)
}

func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, request, rc)
}

type ResolverContext struct {
	// The name of the type this field belongs to
	Object string
	// These are the args after processing, they can be mutated in middleware to change what the resolver will get.
	Args map[string]interface{}
	// The raw field
	Field CollectedField
}

func GetResolverContext(ctx context.Context) *ResolverContext {
	val := ctx.Value(resolver)
	if val == nil {
		return nil
	}

	return val.(*ResolverContext)
}

func WithResolverContext(ctx context.Context, rc *ResolverContext) context.Context {
	return context.WithValue(ctx, resolver, rc)
}

// This is just a convenient wrapper method for CollectFields
func CollectFieldsCtx(ctx context.Context, satisfies []string) []CollectedField {
	reqctx := GetRequestContext(ctx)
	resctx := GetResolverContext(ctx)
	return CollectFields(reqctx.Doc, resctx.Field.Selections, satisfies, reqctx.Variables)
}

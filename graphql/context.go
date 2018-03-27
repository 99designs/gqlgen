package graphql

import (
	"context"

	"github.com/vektah/gqlgen/neelance/errors"
	"github.com/vektah/gqlgen/neelance/query"
)

type RequestContext struct {
	errors.Builder

	Variables map[string]interface{}
	Doc       *query.Document
	Recover   RecoverFunc
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
	return context.WithValue(ctx, request, rc)
}

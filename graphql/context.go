package graphql

import (
	"context"

	"github.com/vektah/gqlgen/neelance/query"
)

type Resolver func(ctx context.Context) (res interface{}, err error)
type ResolverMiddleware func(ctx context.Context, next Resolver) (res interface{}, err error)
type RequestMiddleware func(ctx context.Context, next func(ctx context.Context) []byte) []byte

type RequestContext struct {
	ErrorBuilder

	RawQuery           string
	Variables          map[string]interface{}
	Doc                *query.Document
	Recover            RecoverFunc
	ResolverMiddleware ResolverMiddleware
	RequestMiddleware  RequestMiddleware
}

func DefaultResolverMiddleware(ctx context.Context, next Resolver) (res interface{}, err error) {
	return next(ctx)
}

func DefaultRequestMiddleware(ctx context.Context, next func(ctx context.Context) []byte) []byte {
	return next(ctx)
}

func NewRequestContext(doc *query.Document, query string, variables map[string]interface{}) *RequestContext {
	return &RequestContext{
		Doc:                doc,
		RawQuery:           query,
		Variables:          variables,
		ResolverMiddleware: DefaultResolverMiddleware,
		RequestMiddleware:  DefaultRequestMiddleware,
		Recover:            DefaultRecover,
		ErrorBuilder: ErrorBuilder{
			ErrorPresenter: DefaultErrorPresenter,
		},
	}
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
	// The path of fields to get to this resolver
	Path []interface{}
}

func (r *ResolverContext) PushField(alias string) {
	r.Path = append(r.Path, alias)
}

func (r *ResolverContext) PushIndex(index int) {
	r.Path = append(r.Path, index)
}

func (r *ResolverContext) Pop() {
	r.Path = r.Path[0 : len(r.Path)-1]
}

func GetResolverContext(ctx context.Context) *ResolverContext {
	val := ctx.Value(resolver)
	if val == nil {
		return nil
	}

	return val.(*ResolverContext)
}

func WithResolverContext(ctx context.Context, rc *ResolverContext) context.Context {
	parent := GetResolverContext(ctx)
	rc.Path = nil
	if parent != nil {
		rc.Path = append(rc.Path, parent.Path...)
	}
	if rc.Field.Alias != "" {
		rc.PushField(rc.Field.Alias)
	}
	return context.WithValue(ctx, resolver, rc)
}

// This is just a convenient wrapper method for CollectFields
func CollectFieldsCtx(ctx context.Context, satisfies []string) []CollectedField {
	reqctx := GetRequestContext(ctx)
	resctx := GetResolverContext(ctx)
	return CollectFields(reqctx.Doc, resctx.Field.Selections, satisfies, reqctx.Variables)
}

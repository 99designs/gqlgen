package graphql

import (
	"context"
	"fmt"
	"sync"

	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/gqlerror"
)

type Resolver func(ctx context.Context) (res interface{}, err error)
type FieldMiddleware func(ctx context.Context, next Resolver) (res interface{}, err error)
type RequestMiddleware func(ctx context.Context, next func(ctx context.Context) []byte) []byte

type RequestContext struct {
	RawQuery  string
	Variables map[string]interface{}
	Doc       *ast.QueryDocument
	// ErrorPresenter will be used to generate the error
	// message from errors given to Error().
	ErrorPresenter      ErrorPresenterFunc
	Recover             RecoverFunc
	ResolverMiddleware  FieldMiddleware
	DirectiveMiddleware FieldMiddleware
	RequestMiddleware   RequestMiddleware

	errorsMu sync.Mutex
	Errors   gqlerror.List
}

func DefaultResolverMiddleware(ctx context.Context, next Resolver) (res interface{}, err error) {
	return next(ctx)
}

func DefaultDirectiveMiddleware(ctx context.Context, next Resolver) (res interface{}, err error) {
	return next(ctx)
}

func DefaultRequestMiddleware(ctx context.Context, next func(ctx context.Context) []byte) []byte {
	return next(ctx)
}

func (c *RequestContext) FieldMiddleware(ctx context.Context, next Resolver) interface{} {
	res, err := c.ResolverMiddleware(ctx, next)
	if err != nil {
		c.Error(ctx, err)
		return nil
	}
	return res
}

func NewRequestContext(doc *ast.QueryDocument, query string, variables map[string]interface{}) *RequestContext {
	return &RequestContext{
		Doc:                 doc,
		RawQuery:            query,
		Variables:           variables,
		ResolverMiddleware:  DefaultResolverMiddleware,
		DirectiveMiddleware: DefaultDirectiveMiddleware,
		RequestMiddleware:   DefaultRequestMiddleware,
		Recover:             DefaultRecover,
		ErrorPresenter:      DefaultErrorPresenter,
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
	resctx := GetResolverContext(ctx)
	return CollectFields(ctx, resctx.Field.Selections, satisfies)
}

// Errorf sends an error string to the client, passing it through the formatter.
func (c *RequestContext) Errorf(ctx context.Context, format string, args ...interface{}) {
	c.errorsMu.Lock()
	defer c.errorsMu.Unlock()

	c.Errors = append(c.Errors, c.ErrorPresenter(ctx, fmt.Errorf(format, args...)))
}

// Error sends an error to the client, passing it through the formatter.
func (c *RequestContext) Error(ctx context.Context, err error) {
	c.errorsMu.Lock()
	defer c.errorsMu.Unlock()

	c.Errors = append(c.Errors, c.ErrorPresenter(ctx, err))
}

// AddError is a convenience method for adding an error to the current response
func AddError(ctx context.Context, err error) {
	GetRequestContext(ctx).Error(ctx, err)
}

// AddErrorf is a convenience method for adding an error to the current response
func AddErrorf(ctx context.Context, format string, args ...interface{}) {
	GetRequestContext(ctx).Errorf(ctx, format, args...)
}

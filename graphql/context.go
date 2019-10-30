package graphql

import (
	"context"
	"errors"

	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/gqlerror"
)

type Resolver func(ctx context.Context) (res interface{}, err error)
type ResultMiddleware func(ctx context.Context, next ResponseHandler) *Response
type OperationMiddleware func(ctx context.Context, next OperationHandler, writer Writer)
type FieldMiddleware func(ctx context.Context, next Resolver) (res interface{}, err error)
type ComplexityLimitFunc func(ctx context.Context) int

type RequestContext struct {
	RawQuery      string
	Variables     map[string]interface{}
	OperationName string
	Doc           *ast.QueryDocument

	ComplexityLimit      int
	OperationComplexity  int
	DisableIntrospection bool

	// ErrorPresenter will be used to generate the error
	// message from errors given to Error().
	ErrorPresenter      ErrorPresenterFunc
	Recover             RecoverFunc
	ResolverMiddleware  FieldMiddleware
	DirectiveMiddleware FieldMiddleware
	RequestMiddleware   OperationInterceptor

	Stats Stats
}

func (rc *RequestContext) Validate(ctx context.Context) error {
	if rc.Doc == nil {
		return errors.New("field 'Doc' must be required")
	}
	if rc.RawQuery == "" {
		return errors.New("field 'RawQuery' must be required")
	}
	if rc.Variables == nil {
		rc.Variables = make(map[string]interface{})
	}
	if rc.ResolverMiddleware == nil {
		rc.ResolverMiddleware = DefaultResolverMiddleware
	}
	if rc.DirectiveMiddleware == nil {
		rc.DirectiveMiddleware = DefaultDirectiveMiddleware
	}
	if rc.Recover == nil {
		rc.Recover = DefaultRecover
	}
	if rc.ErrorPresenter == nil {
		rc.ErrorPresenter = DefaultErrorPresenter
	}
	if rc.ComplexityLimit < 0 {
		return errors.New("field 'ComplexityLimit' value must be 0 or more")
	}

	return nil
}

func DefaultResolverMiddleware(ctx context.Context, next Resolver) (res interface{}, err error) {
	return next(ctx)
}

func DefaultDirectiveMiddleware(ctx context.Context, next Resolver) (res interface{}, err error) {
	return next(ctx)
}

type key string

const (
	requestCtx  key = "request_context"
	resolverCtx key = "resolver_context"
)

func GetRequestContext(ctx context.Context) *RequestContext {
	if val, ok := ctx.Value(requestCtx).(*RequestContext); ok {
		return val
	}
	return nil
}

func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, requestCtx, rc)
}

type ResolverContext struct {
	Parent *ResolverContext
	// The name of the type this field belongs to
	Object string
	// These are the args after processing, they can be mutated in middleware to change what the resolver will get.
	Args map[string]interface{}
	// The raw field
	Field CollectedField
	// The index of array in path.
	Index *int
	// The result object of resolver
	Result interface{}
	// IsMethod indicates if the resolver is a method
	IsMethod bool
}

func (r *ResolverContext) Path() []interface{} {
	var path []interface{}
	for it := r; it != nil; it = it.Parent {
		if it.Index != nil {
			path = append(path, *it.Index)
		} else if it.Field.Field != nil {
			path = append(path, it.Field.Alias)
		}
	}

	// because we are walking up the chain, all the elements are backwards, do an inplace flip.
	for i := len(path)/2 - 1; i >= 0; i-- {
		opp := len(path) - 1 - i
		path[i], path[opp] = path[opp], path[i]
	}

	return path
}

func GetResolverContext(ctx context.Context) *ResolverContext {
	if val, ok := ctx.Value(resolverCtx).(*ResolverContext); ok {
		return val
	}
	return nil
}

func WithResolverContext(ctx context.Context, rc *ResolverContext) context.Context {
	rc.Parent = GetResolverContext(ctx)
	return context.WithValue(ctx, resolverCtx, rc)
}

// This is just a convenient wrapper method for CollectFields
func CollectFieldsCtx(ctx context.Context, satisfies []string) []CollectedField {
	resctx := GetResolverContext(ctx)
	return CollectFields(GetRequestContext(ctx), resctx.Field.Selections, satisfies)
}

// CollectAllFields returns a slice of all GraphQL field names that were selected for the current resolver context.
// The slice will contain the unique set of all field names requested regardless of fragment type conditions.
func CollectAllFields(ctx context.Context) []string {
	resctx := GetResolverContext(ctx)
	collected := CollectFields(GetRequestContext(ctx), resctx.Field.Selections, nil)
	uniq := make([]string, 0, len(collected))
Next:
	for _, f := range collected {
		for _, name := range uniq {
			if name == f.Name {
				continue Next
			}
		}
		uniq = append(uniq, f.Name)
	}
	return uniq
}

// Errorf sends an error string to the client, passing it through the formatter.
// Deprecated: use graphql.AddErrorf(ctx, err) instead
func (c *RequestContext) Errorf(ctx context.Context, format string, args ...interface{}) {
	AddErrorf(ctx, format, args...)
}

// Error sends an error to the client, passing it through the formatter.
// Deprecated: use graphql.AddError(ctx, err) instead
func (c *RequestContext) Error(ctx context.Context, err error) {
	AddError(ctx, err)
}

func equalPath(a []interface{}, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

var _ RequestContextMutator = ComplexityLimitFunc(nil)

func (c ComplexityLimitFunc) MutateRequestContext(ctx context.Context, rc *RequestContext) *gqlerror.Error {
	rc.ComplexityLimit = c(ctx)
	return nil
}

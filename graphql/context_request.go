package graphql

import (
	"context"
	"errors"

	"github.com/vektah/gqlparser/ast"
)

type RequestContext struct {
	RawQuery      string
	Variables     map[string]interface{}
	OperationName string
	Doc           *ast.QueryDocument

	DisableIntrospection bool
	Recover              RecoverFunc
	ResolverMiddleware   FieldMiddleware
	DirectiveMiddleware  FieldMiddleware

	Stats Stats
}

const requestCtx key = "request_context"

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

	return nil
}

func GetRequestContext(ctx context.Context) *RequestContext {
	if val, ok := ctx.Value(requestCtx).(*RequestContext); ok {
		return val
	}
	return nil
}

func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, requestCtx, rc)
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

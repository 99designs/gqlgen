package graphql

import (
	"context"
	"errors"

	"github.com/vektah/gqlparser/ast"
)

// Deprecated: Please update all references to OperationContext instead
type RequestContext = OperationContext

type OperationContext struct {
	RawQuery      string
	Variables     map[string]interface{}
	OperationName string
	Doc           *ast.QueryDocument

	Operation            *ast.OperationDefinition
	DisableIntrospection bool
	Recover              RecoverFunc
	ResolverMiddleware   FieldMiddleware
	DirectiveMiddleware  FieldMiddleware

	Stats Stats
}

const operationCtx key = "operation_context"

func (rc *OperationContext) Validate(ctx context.Context) error {
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

// Deprecated: Please update all references to GetOperationContext instead
func GetRequestContext(ctx context.Context) *RequestContext {
	return GetOperationContext(ctx)
}

func GetOperationContext(ctx context.Context) *OperationContext {
	if val, ok := ctx.Value(operationCtx).(*OperationContext); ok {
		return val
	}
	return nil
}

func WithOperationContext(ctx context.Context, rc *OperationContext) context.Context {
	return context.WithValue(ctx, operationCtx, rc)
}

// This is just a convenient wrapper method for CollectFields
func CollectFieldsCtx(ctx context.Context, satisfies []string) []CollectedField {
	resctx := GetResolverContext(ctx)
	return CollectFields(GetOperationContext(ctx), resctx.Field.Selections, satisfies)
}

// CollectAllFields returns a slice of all GraphQL field names that were selected for the current resolver context.
// The slice will contain the unique set of all field names requested regardless of fragment type conditions.
func CollectAllFields(ctx context.Context) []string {
	resctx := GetResolverContext(ctx)
	collected := CollectFields(GetOperationContext(ctx), resctx.Field.Selections, nil)
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
func (c *OperationContext) Errorf(ctx context.Context, format string, args ...interface{}) {
	AddErrorf(ctx, format, args...)
}

// Error sends an error to the client, passing it through the formatter.
// Deprecated: use graphql.AddError(ctx, err) instead
func (c *OperationContext) Error(ctx context.Context, err error) {
	AddError(ctx, err)
}

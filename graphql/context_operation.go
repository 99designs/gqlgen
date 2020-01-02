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

	Stats Stats
}

func (c *OperationContext) Validate(ctx context.Context) error {
	if c.Doc == nil {
		return errors.New("field 'Doc'is required")
	}
	if c.RawQuery == "" {
		return errors.New("field 'RawQuery' is required")
	}
	if c.Variables == nil {
		c.Variables = make(map[string]interface{})
	}
	if c.ResolverMiddleware == nil {
		return errors.New("field 'ResolverMiddleware' is required")
	}
	if c.Recover == nil {
		c.Recover = DefaultRecover
	}

	return nil
}

const operationCtx key = "operation_context"

// Deprecated: Please update all references to GetOperationContext instead
func GetRequestContext(ctx context.Context) *RequestContext {
	return GetOperationContext(ctx)
}

func GetOperationContext(ctx context.Context) *OperationContext {
	if val, ok := ctx.Value(operationCtx).(*OperationContext); ok && val != nil {
		return val
	}
	panic("missing operation context")
}

func WithOperationContext(ctx context.Context, rc *OperationContext) context.Context {
	return context.WithValue(ctx, operationCtx, rc)
}

// This is just a convenient wrapper method for CollectFields
func CollectFieldsCtx(ctx context.Context, satisfies []string) []CollectedField {
	resctx := GetFieldContext(ctx)
	return CollectFields(GetOperationContext(ctx), resctx.Field.Selections, satisfies)
}

// CollectAllFields returns a slice of all GraphQL field names that were selected for the current resolver context.
// The slice will contain the unique set of all field names requested regardless of fragment type conditions.
func CollectAllFields(ctx context.Context) []string {
	resctx := GetFieldContext(ctx)
	collected := CollectFields(GetOperationContext(ctx), resctx.Field.Selections, nil)
	uniq := make(map[string]struct{})
	res := make([]string, 0, len(collected))
	for _, f := range collected {
		if _, ok := uniq[f.Name]; !ok {
			uniq[f.Name] = struct{}{}
			res = append(res, f.Name)
		}

	}
	return res
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

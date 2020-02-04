package extension

import (
	"context"

	"github.com/99designs/gqlgen/graphql/introspection"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/gqlerror"
)

// EnableIntrospection enables clients to reflect all of the types available on the graph.
type Introspection struct {
	AllowFieldFunc func(ctx context.Context, t *introspection.Type, field *introspection.Field) (bool, error)

	AllowInputValueFunc func(ctx context.Context, t *introspection.Type, inputValue *introspection.InputValue) (bool, error)
}

var _ interface {
	graphql.OperationContextMutator
	graphql.HandlerExtension
	graphql.FieldInterceptor
} = Introspection{}

func (c Introspection) ExtensionName() string {
	return "Introspection"
}

func (c Introspection) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func (c Introspection) MutateOperationContext(ctx context.Context, rc *graphql.OperationContext) *gqlerror.Error {
	rc.DisableIntrospection = false
	return nil
}

func (c Introspection) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	res, err = next(ctx)

	fc := graphql.GetFieldContext(ctx)
	t := fc.Parent.Result.(*introspection.Type)
	if fields, ok := res.([]introspection.Field); ok {
		if c.AllowFieldFunc == nil {
			return
		}
		var newFields []introspection.Field
		for _, field := range fields {
			allow, err := c.AllowFieldFunc(ctx, t, &field)
			if err != nil {
				return nil, err
			}
			if allow {
				newFields = append(newFields, field)
			}
		}
		res = newFields
	} else if fields, ok := res.([]introspection.InputValue); ok {
		if c.AllowInputValueFunc == nil {
			return
		}
		var newFields []introspection.InputValue
		for _, field := range fields {
			allow, err := c.AllowInputValueFunc(ctx, t, &field)
			if err != nil {
				return nil, err
			}
			if allow {
				newFields = append(newFields, field)
			}
		}
		res = newFields
	}
	return res, err
}

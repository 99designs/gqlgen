package executor

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
)

// ExtensionList manages a slice of graphql extensions.
type ExtensionList struct {
	es         graphql.ExecutableSchema
	extensions []graphql.HandlerExtension
}

// Extensions creates a new ExtensionList validated with the given schema.
func Extensions(es graphql.ExecutableSchema) *ExtensionList {
	return &ExtensionList{
		es: es,
	}
}

// Extensions returns a copy of this ExtensionList's extensions.
func (l *ExtensionList) Extensions() []graphql.HandlerExtension {
	return append(l.extensions[:0:0], l.extensions...)
}

// Add adds the given extension to this ExtensionList.
func (l *ExtensionList) Add(extension graphql.HandlerExtension) {
	if err := extension.Validate(l.es); err != nil {
		panic(err)
	}

	switch extension.(type) {
	case graphql.OperationParameterMutator,
		graphql.OperationContextMutator,
		graphql.OperationInterceptor,
		graphql.FieldInterceptor,
		graphql.ResponseInterceptor:
		l.extensions = append(l.extensions, extension)

	default:
		panic(fmt.Errorf("cannot Use %T as a gqlgen handler extension because it does not implement any extension hooks", extension))
	}
}

// AroundFields is a convenience method for creating an extension that only implements field middleware
func (l *ExtensionList) AroundFields(f graphql.FieldMiddleware) {
	l.Add(FieldFunc(f))
}

// AroundOperations is a convenience method for creating an extension that only implements operation middleware
func (l *ExtensionList) AroundOperations(f graphql.OperationMiddleware) {
	l.Add(OperationFunc(f))
}

// AroundResponses is a convenience method for creating an extension that only implements response middleware
func (l *ExtensionList) AroundResponses(f graphql.ResponseMiddleware) {
	l.Add(ResponseFunc(f))
}

type OperationFunc func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler

func (r OperationFunc) ExtensionName() string {
	return "InlineOperationFunc"
}

func (r OperationFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return fmt.Errorf("OperationFunc can not be nil")
	}
	return nil
}

func (r OperationFunc) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return r(ctx, next)
}

type ResponseFunc func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response

func (r ResponseFunc) ExtensionName() string {
	return "InlineResponseFunc"
}

func (r ResponseFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return fmt.Errorf("ResponseFunc can not be nil")
	}
	return nil
}

func (r ResponseFunc) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	return r(ctx, next)
}

type FieldFunc func(ctx context.Context, next graphql.Resolver) (res interface{}, err error)

func (f FieldFunc) ExtensionName() string {
	return "InlineFieldFunc"
}

func (f FieldFunc) Validate(schema graphql.ExecutableSchema) error {
	if f == nil {
		return fmt.Errorf("FieldFunc can not be nil")
	}
	return nil
}

func (f FieldFunc) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	return f(ctx, next)
}

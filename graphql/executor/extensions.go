package executor

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
)

// Use adds the given extension to this Executor.
func (e *Executor) Use(extension graphql.HandlerExtension) {
	if err := extension.Validate(e.es); err != nil {
		panic(err)
	}

	switch extension.(type) {
	case graphql.OperationParameterMutator,
		graphql.OperationContextMutator,
		graphql.OperationInterceptor,
		graphql.FieldInterceptor,
		graphql.ResponseInterceptor:
		e.extensions = append(e.extensions, extension)
		e.setExtensions()

	default:
		panic(fmt.Errorf("cannot Use %T as a gqlgen handler extension because it does not implement any extension hooks", extension))
	}
}

// AroundFields is a convenience method for creating an extension that only implements field middleware
func (e *Executor) AroundFields(f graphql.FieldMiddleware) {
	e.Use(FieldFunc(f))
}

// AroundOperations is a convenience method for creating an extension that only implements operation middleware
func (e *Executor) AroundOperations(f graphql.OperationMiddleware) {
	e.Use(OperationFunc(f))
}

// AroundResponses is a convenience method for creating an extension that only implements response middleware
func (e *Executor) AroundResponses(f graphql.ResponseMiddleware) {
	e.Use(ResponseFunc(f))
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

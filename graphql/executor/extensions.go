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
	e.Use(aroundFieldFunc(f))
}

// AroundOperations is a convenience method for creating an extension that only implements operation middleware
func (e *Executor) AroundOperations(f graphql.OperationMiddleware) {
	e.Use(aroundOpFunc(f))
}

// AroundResponses is a convenience method for creating an extension that only implements response middleware
func (e *Executor) AroundResponses(f graphql.ResponseMiddleware) {
	e.Use(aroundRespFunc(f))
}

func (e *Executor) setExtensions() {
	e.operationMiddleware = func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		return next(ctx)
	}
	e.responseMiddleware = func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		return next(ctx)
	}
	e.fieldMiddleware = func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		return next(ctx)
	}

	// this loop goes backwards so the first extension is the outer most middleware and runs first.
	for i := len(e.extensions) - 1; i >= 0; i-- {
		p := e.extensions[i]
		if p, ok := p.(graphql.OperationInterceptor); ok {
			previous := e.operationMiddleware
			e.operationMiddleware = func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
				return p.InterceptOperation(ctx, func(ctx context.Context) graphql.ResponseHandler {
					return previous(ctx, next)
				})
			}
		}

		if p, ok := p.(graphql.ResponseInterceptor); ok {
			previous := e.responseMiddleware
			e.responseMiddleware = func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
				return p.InterceptResponse(ctx, func(ctx context.Context) *graphql.Response {
					return previous(ctx, next)
				})
			}
		}

		if p, ok := p.(graphql.FieldInterceptor); ok {
			previous := e.fieldMiddleware
			e.fieldMiddleware = func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
				return p.InterceptField(ctx, func(ctx context.Context) (res interface{}, err error) {
					return previous(ctx, next)
				})
			}
		}
	}

	for _, p := range e.extensions {
		if p, ok := p.(graphql.OperationParameterMutator); ok {
			e.operationParameterMutators = append(e.operationParameterMutators, p)
		}

		if p, ok := p.(graphql.OperationContextMutator); ok {
			e.operationContextMutators = append(e.operationContextMutators, p)
		}
	}
}

type aroundOpFunc func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler

func (r aroundOpFunc) ExtensionName() string {
	return "InlineOperationFunc"
}

func (r aroundOpFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return fmt.Errorf("OperationFunc can not be nil")
	}
	return nil
}

func (r aroundOpFunc) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return r(ctx, next)
}

type aroundRespFunc func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response

func (r aroundRespFunc) ExtensionName() string {
	return "InlineResponseFunc"
}

func (r aroundRespFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return fmt.Errorf("ResponseFunc can not be nil")
	}
	return nil
}

func (r aroundRespFunc) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	return r(ctx, next)
}

type aroundFieldFunc func(ctx context.Context, next graphql.Resolver) (res interface{}, err error)

func (f aroundFieldFunc) ExtensionName() string {
	return "InlineFieldFunc"
}

func (f aroundFieldFunc) Validate(schema graphql.ExecutableSchema) error {
	if f == nil {
		return fmt.Errorf("FieldFunc can not be nil")
	}
	return nil
}

func (f aroundFieldFunc) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	return f(ctx, next)
}

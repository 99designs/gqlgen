package graphql

import (
	"context"
)

var _ Tracer = (*NopTracer)(nil)

type Tracer interface {
	StartOperationExecution(ctx context.Context) context.Context
	StartFieldExecution(ctx context.Context, field CollectedField) context.Context
	StartFieldResolverExecution(ctx context.Context, rc *ResolverContext) context.Context
	StartFieldChildExecution(ctx context.Context) context.Context
	EndFieldExecution(ctx context.Context)
	EndOperationExecution(ctx context.Context)
}

type NopTracer struct{}

func (NopTracer) StartOperationExecution(ctx context.Context) context.Context {
	return ctx
}

func (NopTracer) StartFieldExecution(ctx context.Context, field CollectedField) context.Context {
	return ctx
}

func (NopTracer) StartFieldResolverExecution(ctx context.Context, rc *ResolverContext) context.Context {
	return ctx
}

func (NopTracer) StartFieldChildExecution(ctx context.Context) context.Context {
	return ctx
}

func (NopTracer) EndFieldExecution(ctx context.Context) {
}

func (NopTracer) EndOperationExecution(ctx context.Context) {
}

package graphql

import "context"

var _ Tracer = (*NopTracer)(nil)

type Tracer interface {
	StartOperationExecution(ctx context.Context) context.Context
	EndOperationExecution(ctx context.Context)
	StartFieldExecution(ctx context.Context) context.Context
	EndFieldExecution(ctx context.Context)
}

type NopTracer struct{}

func (NopTracer) StartOperationExecution(ctx context.Context) context.Context {
	return ctx
}

func (NopTracer) EndOperationExecution(ctx context.Context) {
}

func (NopTracer) StartFieldExecution(ctx context.Context) context.Context {
	return ctx
}

func (NopTracer) EndFieldExecution(ctx context.Context) {
}

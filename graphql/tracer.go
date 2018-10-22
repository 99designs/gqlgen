package graphql

import "context"

var _ Tracer = (*NopTracer)(nil)

type Tracer interface {
	StartRequestTracing(ctx context.Context) context.Context
	EndRequestTracing(ctx context.Context)
	StartFieldTracing(ctx context.Context) context.Context
	EndFieldTracing(ctx context.Context)
}

type NopTracer struct{}

func (NopTracer) StartRequestTracing(ctx context.Context) context.Context {
	return ctx
}

func (NopTracer) EndRequestTracing(ctx context.Context) {
}

func (NopTracer) StartFieldTracing(ctx context.Context) context.Context {
	return ctx
}

func (NopTracer) EndFieldTracing(ctx context.Context) {
}

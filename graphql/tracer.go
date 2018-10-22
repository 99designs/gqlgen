package graphql

import "context"

var _ Tracer = (*NopTracer)(nil)

type Tracer interface {
	StartFieldTracing(ctx context.Context) (context.Context, error)
	EndFieldTracing(ctx context.Context) error
}

type NopTracer struct{}

func (NopTracer) StartFieldTracing(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (NopTracer) EndFieldTracing(ctx context.Context) error {
	return nil
}

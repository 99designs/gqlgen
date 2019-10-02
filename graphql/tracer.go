package graphql

import (
	"context"
)

var _ Tracer = (*NopTracer)(nil)

type Tracer interface {
	StartOperationParsing(ctx context.Context) context.Context
	EndOperationParsing(ctx context.Context)
	StartOperationValidation(ctx context.Context) context.Context
	EndOperationValidation(ctx context.Context)
	StartOperationExecution(ctx context.Context) context.Context
	StartFieldExecution(ctx context.Context, field CollectedField) context.Context
	StartFieldResolverExecution(ctx context.Context, rc *ResolverContext) context.Context
	StartFieldChildExecution(ctx context.Context) context.Context
	EndFieldExecution(ctx context.Context)
	EndOperationExecution(ctx context.Context)
}

type NopTracer struct{}

func (NopTracer) StartOperationParsing(ctx context.Context) context.Context {
	return ctx
}

func (NopTracer) EndOperationParsing(ctx context.Context) {
}

func (NopTracer) StartOperationValidation(ctx context.Context) context.Context {
	return ctx
}

func (NopTracer) EndOperationValidation(ctx context.Context) {
}

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

type tracerWrapper struct {
	tracer1 Tracer
	tracer2 Tracer
}

func (tw *tracerWrapper) StartOperationParsing(ctx context.Context) context.Context {
	ctx = tw.tracer1.StartOperationParsing(ctx)
	ctx = tw.tracer2.StartOperationParsing(ctx)
	return ctx
}

func (tw *tracerWrapper) EndOperationParsing(ctx context.Context) {
	tw.tracer2.EndOperationParsing(ctx)
	tw.tracer1.EndOperationParsing(ctx)
}

func (tw *tracerWrapper) StartOperationValidation(ctx context.Context) context.Context {
	ctx = tw.tracer1.StartOperationValidation(ctx)
	ctx = tw.tracer2.StartOperationValidation(ctx)
	return ctx
}

func (tw *tracerWrapper) EndOperationValidation(ctx context.Context) {
	tw.tracer2.EndOperationValidation(ctx)
	tw.tracer1.EndOperationValidation(ctx)
}

func (tw *tracerWrapper) StartOperationExecution(ctx context.Context) context.Context {
	ctx = tw.tracer1.StartOperationExecution(ctx)
	ctx = tw.tracer2.StartOperationExecution(ctx)
	return ctx
}

func (tw *tracerWrapper) StartFieldExecution(ctx context.Context, field CollectedField) context.Context {
	ctx = tw.tracer1.StartFieldExecution(ctx, field)
	ctx = tw.tracer2.StartFieldExecution(ctx, field)
	return ctx
}

func (tw *tracerWrapper) StartFieldResolverExecution(ctx context.Context, rc *ResolverContext) context.Context {
	ctx = tw.tracer1.StartFieldResolverExecution(ctx, rc)
	ctx = tw.tracer2.StartFieldResolverExecution(ctx, rc)
	return ctx
}

func (tw *tracerWrapper) StartFieldChildExecution(ctx context.Context) context.Context {
	ctx = tw.tracer1.StartFieldChildExecution(ctx)
	ctx = tw.tracer2.StartFieldChildExecution(ctx)
	return ctx
}

func (tw *tracerWrapper) EndFieldExecution(ctx context.Context) {
	tw.tracer2.EndFieldExecution(ctx)
	tw.tracer1.EndFieldExecution(ctx)
}

func (tw *tracerWrapper) EndOperationExecution(ctx context.Context) {
	tw.tracer2.EndOperationExecution(ctx)
	tw.tracer1.EndOperationExecution(ctx)
}

package gqlapollotracing

import (
	"context"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

var _ graphql.Tracer = (*tracerImpl)(nil)

func NewTracer() graphql.Tracer {
	return &tracerImpl{}
}

var timeNowFunc = time.Now

var ctxTracingKey = &struct{ tmp string }{}
var ctxExecSpanKey = &struct{ tmp string }{}

type tracerImpl struct {
}

func getTracingData(ctx context.Context) *tracingData {
	return ctx.Value(ctxTracingKey).(*tracingData)
}

func getExecutionSpan(ctx context.Context) *executionSpan {
	return ctx.Value(ctxExecSpanKey).(*executionSpan)
}

func (t *tracerImpl) StartOperationParsing(ctx context.Context) context.Context {
	now := timeNowFunc()
	td := &tracingData{
		StartTime: now,
		Parsing: &startOffset{
			StartTime: now,
		},
	}
	ctx = context.WithValue(ctx, ctxTracingKey, td)
	return ctx
}

func (t *tracerImpl) EndOperationParsing(ctx context.Context) {
	td := getTracingData(ctx)
	td.Parsing.EndTime = timeNowFunc()
}

func (t *tracerImpl) StartOperationValidation(ctx context.Context) context.Context {
	td := getTracingData(ctx)
	td.Validation = &startOffset{}
	td.Validation.StartTime = timeNowFunc()
	return ctx
}

func (t *tracerImpl) EndOperationValidation(ctx context.Context) {
	td := getTracingData(ctx)
	td.Validation.EndTime = timeNowFunc()
}

func (t *tracerImpl) StartOperationExecution(ctx context.Context) context.Context {
	return ctx
}

func (t *tracerImpl) StartFieldExecution(ctx context.Context, field graphql.CollectedField) context.Context {

	td := getTracingData(ctx)
	es := &executionSpan{
		startOffset: startOffset{
			StartTime: timeNowFunc(),
		},
		ParentType: field.ObjectDefinition.Name,
		FieldName:  field.Name,
		ReturnType: field.Definition.Type.String(),
	}
	ctx = context.WithValue(ctx, ctxExecSpanKey, es)
	td.mu.Lock()
	defer td.mu.Unlock()
	if td.Execution == nil {
		td.Execution = &execution{}
	}
	td.Execution.Resolvers = append(td.Execution.Resolvers, es)

	return ctx
}

func (t *tracerImpl) StartFieldResolverExecution(ctx context.Context, rc *graphql.ResolverContext) context.Context {
	es := getExecutionSpan(ctx)
	es.Path = rc.Path()

	return ctx
}

func (t *tracerImpl) StartFieldChildExecution(ctx context.Context) context.Context {
	return ctx
}

func (t *tracerImpl) EndFieldExecution(ctx context.Context) {
	es := getExecutionSpan(ctx)
	es.EndTime = timeNowFunc()
}

func (t *tracerImpl) EndOperationExecution(ctx context.Context) {
	td := getTracingData(ctx)
	td.EndTime = timeNowFunc()
	td.prepare()
}

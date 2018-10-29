package gqlopencensus

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"go.opencensus.io/trace"
)

var _ graphql.Tracer = (*tracerImpl)(nil)

// New returns Tracer for OpenCensus.
// see https://go.opencensus.io/trace
func New(opts ...Option) graphql.Tracer {
	tracer := &tracerImpl{}
	cfg := &config{tracer}

	for _, opt := range opts {
		opt.apply(cfg)
	}

	return tracer
}

type tracerImpl struct {
	startOperationExecutions     []func(ctx context.Context) context.Context
	startFieldExecutions         []func(ctx context.Context, field graphql.CollectedField) context.Context
	startFieldResolverExecutions []func(ctx context.Context, rc *graphql.ResolverContext) context.Context
	startFieldChildExecutions    []func(ctx context.Context) context.Context
	endFieldExecutions           []func(ctx context.Context)
	endOperationExecutions       []func(ctx context.Context)
}

func (t *tracerImpl) StartOperationExecution(ctx context.Context) context.Context {
	ctx, span := trace.StartSpan(ctx, operationName(ctx))
	if !span.IsRecordingEvents() {
		return ctx
	}
	requestContext := graphql.GetRequestContext(ctx)
	span.AddAttributes(
		trace.StringAttribute("request.query", requestContext.RawQuery),
	)
	for key, val := range requestContext.Variables {
		span.AddAttributes(
			trace.StringAttribute(fmt.Sprintf("request.variables.%s", key), fmt.Sprintf("%+v", val)),
		)
	}
	for _, f := range t.startOperationExecutions {
		ctx = f(ctx)
	}

	return ctx
}

func (t *tracerImpl) StartFieldExecution(ctx context.Context, field graphql.CollectedField) context.Context {
	ctx, span := trace.StartSpan(ctx, field.ObjectDefinition.Name+"/"+field.Name)
	if !span.IsRecordingEvents() {
		return ctx
	}
	span.AddAttributes(
		trace.StringAttribute("resolver.object", field.ObjectDefinition.Name),
		trace.StringAttribute("resolver.field", field.Name),
		trace.StringAttribute("resolver.alias", field.Alias),
	)
	for _, arg := range field.Arguments {
		if arg.Value != nil {
			span.AddAttributes(
				trace.StringAttribute(fmt.Sprintf("resolver.args.%s", arg.Name), arg.Value.String()),
			)
		}
	}

	for _, f := range t.startFieldExecutions {
		ctx = f(ctx, field)
	}

	return ctx
}

func (t *tracerImpl) StartFieldResolverExecution(ctx context.Context, rc *graphql.ResolverContext) context.Context {
	span := trace.FromContext(ctx)
	if !span.IsRecordingEvents() {
		return ctx
	}
	span.AddAttributes(
		trace.StringAttribute("resolver.path", fmt.Sprintf("%+v", rc.Path())),
	)
	for _, f := range t.startFieldResolverExecutions {
		ctx = f(ctx, rc)
	}

	return ctx
}

func (t *tracerImpl) StartFieldChildExecution(ctx context.Context) context.Context {
	span := trace.FromContext(ctx)
	if !span.IsRecordingEvents() {
		return ctx
	}
	for _, f := range t.startFieldChildExecutions {
		ctx = f(ctx)
	}
	return ctx
}

func (t *tracerImpl) EndFieldExecution(ctx context.Context) {
	span := trace.FromContext(ctx)
	defer span.End()
	if !span.IsRecordingEvents() {
		return
	}
	for _, f := range t.endFieldExecutions {
		f(ctx)
	}
}

func (t *tracerImpl) EndOperationExecution(ctx context.Context) {
	span := trace.FromContext(ctx)
	defer span.End()
	if !span.IsRecordingEvents() {
		return
	}
	for _, f := range t.endOperationExecutions {
		f(ctx)
	}
}

func operationName(ctx context.Context) string {
	requestContext := graphql.GetRequestContext(ctx)
	requestName := "nameless-operation"
	if requestContext.Doc != nil && len(requestContext.Doc.Operations) != 0 {
		op := requestContext.Doc.Operations[0]
		if op.Name != "" {
			requestName = op.Name
		}
	}

	return requestName
}

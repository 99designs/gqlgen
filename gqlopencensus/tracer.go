package gqlopencensus

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"go.opencensus.io/trace"
)

var _ graphql.Tracer = (tracerImpl)(0)

// New returns Tracer for OpenCensus.
// see https://go.opencensus.io/trace
func New(opts ...Option) graphql.Tracer {
	var tracer tracerImpl
	cfg := &config{tracer}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg.tracer
}

type tracerImpl int

func (tracerImpl) StartOperationParsing(ctx context.Context) context.Context {
	return ctx
}

func (tracerImpl) EndOperationParsing(ctx context.Context) {
}

func (tracerImpl) StartOperationValidation(ctx context.Context) context.Context {
	return ctx
}

func (tracerImpl) EndOperationValidation(ctx context.Context) {
}

func (tracerImpl) StartOperationExecution(ctx context.Context) context.Context {
	ctx, span := trace.StartSpan(ctx, operationName(ctx))
	if !span.IsRecordingEvents() {
		return ctx
	}
	requestContext := graphql.GetRequestContext(ctx)
	span.AddAttributes(
		trace.StringAttribute("request.query", requestContext.RawQuery),
	)
	if requestContext.ComplexityLimit > 0 {
		span.AddAttributes(
			trace.Int64Attribute("request.complexityLimit", int64(requestContext.ComplexityLimit)),
			trace.Int64Attribute("request.operationComplexity", int64(requestContext.OperationComplexity)),
		)
	}

	for key, val := range requestContext.Variables {
		span.AddAttributes(
			trace.StringAttribute(fmt.Sprintf("request.variables.%s", key), fmt.Sprintf("%+v", val)),
		)
	}

	return ctx
}

func (tracerImpl) StartFieldExecution(ctx context.Context, field graphql.CollectedField) context.Context {
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

	return ctx
}

func (tracerImpl) StartFieldResolverExecution(ctx context.Context, rc *graphql.ResolverContext) context.Context {
	span := trace.FromContext(ctx)
	if !span.IsRecordingEvents() {
		return ctx
	}
	span.AddAttributes(
		trace.StringAttribute("resolver.path", fmt.Sprintf("%+v", rc.Path())),
	)

	return ctx
}

func (tracerImpl) StartFieldChildExecution(ctx context.Context) context.Context {
	span := trace.FromContext(ctx)
	if !span.IsRecordingEvents() {
		return ctx
	}
	return ctx
}

func (tracerImpl) EndFieldExecution(ctx context.Context) {
	span := trace.FromContext(ctx)
	defer span.End()
	if !span.IsRecordingEvents() {
		return
	}
}

func (tracerImpl) EndOperationExecution(ctx context.Context) {
	span := trace.FromContext(ctx)
	defer span.End()
	if !span.IsRecordingEvents() {
		return
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

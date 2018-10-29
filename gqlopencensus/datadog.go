package gqlopencensus

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"go.opencensus.io/trace"
)

// WithDataDog provides DataDog specific span attrs.
// see github.com/DataDog/opencensus-go-exporter-datadog
func WithDataDog() Option {
	return WithStartFieldResolverExecution(func(ctx context.Context, rc *graphql.ResolverContext) context.Context {
		span := trace.FromContext(ctx)
		span.AddAttributes(
			// key from gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext#ResourceName
			trace.StringAttribute("resource.name", operationName(ctx)),
		)
		return ctx
	})
}

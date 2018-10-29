package gqlopencensus_test

import (
	"context"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/gqlopencensus"
	"github.com/99designs/gqlgen/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/ast"
	"go.opencensus.io/trace"
)

func TestTracer(t *testing.T) {
	var logs []string
	var mu sync.Mutex

	tracer := gqlopencensus.New(
		gqlopencensus.WithStartOperationExecution(func(ctx context.Context) context.Context {
			logs = append(logs, "StartOperationExecution")
			return ctx
		}),
		gqlopencensus.WithStartFieldExecution(func(ctx context.Context, field graphql.CollectedField) context.Context {
			logs = append(logs, "StartFieldExecution")
			return ctx
		}),
		gqlopencensus.WithStartFieldResolverExecution(func(ctx context.Context, rc *graphql.ResolverContext) context.Context {
			logs = append(logs, "StartFieldResolverExecution")
			return ctx
		}),
		gqlopencensus.WithStartFieldChildExecution(func(ctx context.Context) context.Context {
			logs = append(logs, "StartFieldChildExecution")
			return ctx
		}),
		gqlopencensus.WithEndFieldExecution(func(ctx context.Context) {
			logs = append(logs, "EndFieldExecution")
		}),
		gqlopencensus.WithEndOperationExecutions(func(ctx context.Context) {
			logs = append(logs, "EndOperationExecution")
		}),
	)

	specs := []struct {
		SpecName string
		Sampler  trace.Sampler
		Expected []string
	}{
		{
			SpecName: "with sampling",
			Sampler:  trace.AlwaysSample(),
			Expected: []string{
				"StartOperationExecution",
				"StartFieldExecution",
				"StartFieldResolverExecution",
				"StartFieldChildExecution",
				"EndFieldExecution",
				"EndOperationExecution",
			},
		},
		{
			SpecName: "without sampling",
			Sampler:  trace.NeverSample(),
			Expected: nil,
		},
	}

	for _, spec := range specs {
		t.Run(spec.SpecName, func(t *testing.T) {
			mu.Lock()
			defer mu.Unlock()
			logs = nil

			ctx := context.Background()
			ctx = graphql.WithRequestContext(ctx, &graphql.RequestContext{})
			ctx, _ = trace.StartSpan(ctx, "test", trace.WithSampler(spec.Sampler))
			ctx = tracer.StartOperationExecution(ctx)
			ctx = tracer.StartFieldExecution(ctx, graphql.CollectedField{
				Field: &ast.Field{
					Name: "F",
					ObjectDefinition: &ast.Definition{
						Name: "OD",
					},
				},
			})
			ctx = tracer.StartFieldResolverExecution(ctx, &graphql.ResolverContext{})
			ctx = tracer.StartFieldChildExecution(ctx)
			tracer.EndFieldExecution(ctx)
			tracer.EndOperationExecution(ctx)

			assert.Equal(t, spec.Expected, logs)
		})
	}
}

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

var _ trace.Exporter = (*testExporter)(nil)

type testExporter struct {
	sync.Mutex

	Spans []*trace.SpanData
}

func (te *testExporter) ExportSpan(s *trace.SpanData) {
	te.Lock()
	defer te.Unlock()

	te.Spans = append(te.Spans, s)
}

func (te *testExporter) Reset() {
	te.Lock()
	defer te.Unlock()

	te.Spans = nil
}

func TestTracer(t *testing.T) {
	var mu sync.Mutex

	exporter := &testExporter{}

	trace.RegisterExporter(exporter)
	defer trace.UnregisterExporter(exporter)

	specs := []struct {
		SpecName      string
		Tracer        graphql.Tracer
		Sampler       trace.Sampler
		ExpectedAttrs []map[string]interface{}
	}{
		{
			SpecName: "with sampling",
			Tracer:   gqlopencensus.New(),
			Sampler:  trace.AlwaysSample(),
			ExpectedAttrs: []map[string]interface{}{
				{
					"resolver.object": "OD",
					"resolver.field":  "F",
					"resolver.alias":  "F",
					"resolver.path":   "[]",
				},
				{
					"request.query":               "query { foobar }",
					"request.variables.fizz":      "buzz",
					"request.complexityLimit":     int64(1000),
					"request.operationComplexity": int64(100),
				},
			},
		},
		{
			SpecName:      "without sampling",
			Tracer:        gqlopencensus.New(),
			Sampler:       trace.NeverSample(),
			ExpectedAttrs: nil,
		},
		{
			SpecName: "with sampling & DataDog",
			Tracer:   gqlopencensus.New(gqlopencensus.WithDataDog()),
			Sampler:  trace.AlwaysSample(),
			ExpectedAttrs: []map[string]interface{}{
				{
					"resolver.object": "OD",
					"resolver.field":  "F",
					"resolver.alias":  "F",
					"resolver.path":   "[]",
					"resource.name":   "nameless-operation",
				},
				{
					"request.query":               "query { foobar }",
					"request.variables.fizz":      "buzz",
					"request.complexityLimit":     int64(1000),
					"request.operationComplexity": int64(100),
				},
			},
		},
		{
			SpecName:      "without sampling & DataDog",
			Tracer:        gqlopencensus.New(gqlopencensus.WithDataDog()),
			Sampler:       trace.NeverSample(),
			ExpectedAttrs: nil,
		},
	}

	for _, spec := range specs {
		t.Run(spec.SpecName, func(t *testing.T) {
			mu.Lock()
			defer mu.Unlock()
			exporter.Reset()

			tracer := spec.Tracer
			ctx := context.Background()
			ctx = graphql.WithRequestContext(ctx, &graphql.RequestContext{
				RawQuery: "query { foobar }",
				Variables: map[string]interface{}{
					"fizz": "buzz",
				},
				ComplexityLimit:     1000,
				OperationComplexity: 100,
			})
			ctx, _ = trace.StartSpan(ctx, "test", trace.WithSampler(spec.Sampler))
			ctx = tracer.StartOperationExecution(ctx)
			{
				ctx2 := tracer.StartFieldExecution(ctx, graphql.CollectedField{
					Field: &ast.Field{
						Name:  "F",
						Alias: "F",
						ObjectDefinition: &ast.Definition{
							Name: "OD",
						},
					},
				})
				ctx2 = tracer.StartFieldResolverExecution(ctx2, &graphql.ResolverContext{})
				ctx2 = tracer.StartFieldChildExecution(ctx2)
				tracer.EndFieldExecution(ctx2)
			}
			tracer.EndOperationExecution(ctx)

			if len(spec.ExpectedAttrs) == 0 && len(exporter.Spans) != 0 {
				t.Errorf("unexpected spans: %+v", exporter.Spans)
			} else if len(spec.ExpectedAttrs) != len(exporter.Spans) {
				assert.Equal(t, len(spec.ExpectedAttrs), len(exporter.Spans))
			} else {
				for idx, expectedAttrs := range spec.ExpectedAttrs {
					span := exporter.Spans[idx]
					assert.Equal(t, expectedAttrs, span.Attributes)
				}
			}
		})
	}
}

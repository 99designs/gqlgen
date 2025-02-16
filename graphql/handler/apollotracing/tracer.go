package apollotracing

import (
	"context"
	"sync"
	"time"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

type (
	Tracer struct{}

	TracingExtension struct {
		mu         sync.Mutex
		Version    int           `json:"version"`
		StartTime  time.Time     `json:"startTime"`
		EndTime    time.Time     `json:"endTime"`
		Duration   time.Duration `json:"duration"`
		Parsing    Span          `json:"parsing"`
		Validation Span          `json:"validation"`
		Execution  struct {
			Resolvers []*ResolverExecution `json:"resolvers"`
		} `json:"execution"`
	}

	Span struct {
		StartOffset time.Duration `json:"startOffset"`
		Duration    time.Duration `json:"duration"`
	}

	ResolverExecution struct {
		Path        ast.Path      `json:"path"`
		ParentType  string        `json:"parentType"`
		FieldName   string        `json:"fieldName"`
		ReturnType  string        `json:"returnType"`
		StartOffset time.Duration `json:"startOffset"`
		Duration    time.Duration `json:"duration"`
	}
)

var _ interface {
	graphql.HandlerExtension
	graphql.ResponseInterceptor
	graphql.FieldInterceptor
} = Tracer{}

func (Tracer) ExtensionName() string {
	return "ApolloTracing"
}

func (Tracer) Validate(graphql.ExecutableSchema) error {
	return nil
}

func (Tracer) InterceptField(ctx context.Context, next graphql.Resolver) (any, error) {
	td, ok := graphql.GetExtension(ctx, "tracing").(*TracingExtension)
	if !ok {
		return next(ctx)
	}

	start := graphql.Now()

	defer func() {
		end := graphql.Now()

		opCtx := graphql.GetOperationContext(ctx)
		fc := graphql.GetFieldContext(ctx)
		resolver := &ResolverExecution{
			Path:        fc.Path(),
			ParentType:  fc.Object,
			FieldName:   fc.Field.Name,
			ReturnType:  fc.Field.Definition.Type.String(),
			StartOffset: start.Sub(opCtx.Stats.OperationStart),
			Duration:    end.Sub(start),
		}

		td.mu.Lock()
		td.Execution.Resolvers = append(td.Execution.Resolvers, resolver)
		td.mu.Unlock()
	}()

	return next(ctx)
}

func (Tracer) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	if !graphql.HasOperationContext(ctx) {
		return next(ctx)
	}

	opCtx := graphql.GetOperationContext(ctx)

	start := opCtx.Stats.OperationStart

	td := &TracingExtension{
		Version:   1,
		StartTime: start,
		Parsing: Span{
			StartOffset: opCtx.Stats.Parsing.Start.Sub(start),
			Duration:    opCtx.Stats.Parsing.End.Sub(opCtx.Stats.Parsing.Start),
		},

		Validation: Span{
			StartOffset: opCtx.Stats.Validation.Start.Sub(start),
			Duration:    opCtx.Stats.Validation.End.Sub(opCtx.Stats.Validation.Start),
		},
	}

	graphql.RegisterExtension(ctx, "tracing", td)
	resp := next(ctx)

	end := graphql.Now()
	td.EndTime = end
	td.Duration = end.Sub(start)

	return resp
}

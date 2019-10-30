package apollotracing

import (
	"context"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

type (
	ApolloTracing struct{}

	TracingExtension struct {
		mu         sync.Mutex    `json:"-"`
		Version    int           `json:"version"`
		StartTime  time.Time     `json:"startTime"`
		EndTime    time.Time     `json:"endTime"`
		Duration   time.Duration `json:"duration"`
		Parsing    Span          `json:"parsing"`
		Validation Span          `json:"validation"`
		Execution  struct {
			Resolvers []ResolverExecution `json:"resolvers"`
		} `json:"execution"`
	}

	Span struct {
		StartOffset time.Duration `json:"startOffset"`
		Duration    time.Duration `json:"duration"`
	}

	ResolverExecution struct {
		Path        []interface{} `json:"path"`
		ParentType  string        `json:"parentType"`
		FieldName   string        `json:"fieldName"`
		ReturnType  string        `json:"returnType"`
		StartOffset time.Duration `json:"startOffset"`
		Duration    time.Duration `json:"duration"`
	}
)

var _ graphql.ResponseInterceptor = ApolloTracing{}
var _ graphql.FieldInterceptor = ApolloTracing{}

func New() graphql.HandlerPlugin {
	return &ApolloTracing{}
}

func (a ApolloTracing) InterceptField(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	rc := graphql.GetRequestContext(ctx)
	td, ok := graphql.GetExtension(ctx, "tracing").(*TracingExtension)
	if !ok {
		panic("missing tracing extension")
	}

	start := graphql.Now()

	defer func() {
		td.mu.Lock()
		defer td.mu.Unlock()
		fc := graphql.GetResolverContext(ctx)

		end := graphql.Now()

		td.Execution.Resolvers = append(td.Execution.Resolvers, ResolverExecution{
			Path:        fc.Path(),
			ParentType:  fc.Object,
			FieldName:   fc.Field.Name,
			ReturnType:  fc.Field.Definition.Type.String(),
			StartOffset: start.Sub(rc.Stats.OperationStart),
			Duration:    end.Sub(start),
		})
	}()

	return next(ctx)
}

func (a ApolloTracing) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	rc := graphql.GetRequestContext(ctx)

	start := rc.Stats.OperationStart

	td := &TracingExtension{
		Version:   1,
		StartTime: start,
		Parsing: Span{
			StartOffset: rc.Stats.Parsing.Start.Sub(start),
			Duration:    rc.Stats.Parsing.End.Sub(rc.Stats.Parsing.Start),
		},

		Validation: Span{
			StartOffset: rc.Stats.Validation.Start.Sub(start),
			Duration:    rc.Stats.Validation.End.Sub(rc.Stats.Validation.Start),
		},

		Execution: struct {
			Resolvers []ResolverExecution `json:"resolvers"`
		}{},
	}

	graphql.RegisterExtension(ctx, "tracing", td)

	resp := next(ctx)

	end := graphql.Now()
	td.EndTime = end
	td.Duration = end.Sub(start)

	return resp
}

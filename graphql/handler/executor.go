package handler

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/ast"
	"github.com/vektah/gqlparser/gqlerror"
	"github.com/vektah/gqlparser/parser"
	"github.com/vektah/gqlparser/validator"
)

type executor struct {
	handler                graphql.Handler
	es                     graphql.ExecutableSchema
	requestMutators        []graphql.RequestMutator
	requestContextMutators []graphql.RequestContextMutator
}

var _ graphql.GraphExecutor = executor{}

func newExecutor(es graphql.ExecutableSchema, plugins []graphql.HandlerPlugin) executor {
	e := executor{
		es: es,
	}
	handler := e.executableSchemaHandler
	// this loop goes backwards so the first plugin is the outer most middleware and runs first.
	for i := len(plugins) - 1; i >= 0; i-- {
		p := plugins[i]
		if p, ok := p.(graphql.RequestMiddleware); ok {
			handler = p.InterceptRequest(handler)
		}
	}

	for _, p := range plugins {

		if p, ok := p.(graphql.RequestMutator); ok {
			e.requestMutators = append(e.requestMutators, p)
		}

		if p, ok := p.(graphql.RequestContextMutator); ok {
			e.requestContextMutators = append(e.requestContextMutators, p)
		}
	}

	e.handler = handler

	return e
}

func (e executor) DispatchRequest(ctx context.Context, writer graphql.Writer) {
	e.handler(ctx, writer)
}

func (e executor) CreateRequestContext(ctx context.Context, params *graphql.RawParams) (*graphql.RequestContext, gqlerror.List) {
	for _, p := range e.requestMutators {
		if err := p.MutateRequest(ctx, params); err != nil {
			return nil, gqlerror.List{err}
		}
	}

	var gerr *gqlerror.Error

	rc := &graphql.RequestContext{
		DisableIntrospection: true,
		Recover:              graphql.DefaultRecover,
		ErrorPresenter:       graphql.DefaultErrorPresenter,
		ResolverMiddleware:   nil,
		RequestMiddleware:    nil,
		Tracer:               graphql.NopTracer{},
		ComplexityLimit:      0,
		RawQuery:             params.Query,
		OperationName:        params.OperationName,
		Variables:            params.Variables,
		Extensions:           params.Extensions,
	}

	rc.Doc, gerr = e.parseOperation(ctx, rc)
	if gerr != nil {
		return nil, []*gqlerror.Error{gerr}
	}

	ctx, op, listErr := e.validateOperation(ctx, rc)
	if len(listErr) != 0 {
		return nil, listErr
	}

	vars, err := validator.VariableValues(e.es.Schema(), op, rc.Variables)
	if err != nil {
		return nil, gqlerror.List{err}
	}

	rc.Variables = vars

	for _, p := range e.requestContextMutators {
		if err := p.MutateRequestContext(ctx, rc); err != nil {
			return nil, gqlerror.List{err}
		}
	}

	return rc, nil
}

// executableSchemaHandler is the inner most handler, it invokes the graph directly after all middleware
// and sends responses to the transport so it can be returned to the client
func (e *executor) executableSchemaHandler(ctx context.Context, write graphql.Writer) {
	rc := graphql.GetRequestContext(ctx)

	op := rc.Doc.Operations.ForName(rc.OperationName)

	switch op.Operation {
	case ast.Query:
		resp := e.es.Query(ctx, op)

		write(getStatus(resp), resp)
	case ast.Mutation:
		resp := e.es.Mutation(ctx, op)
		write(getStatus(resp), resp)
	case ast.Subscription:
		resp := e.es.Subscription(ctx, op)

		for w := resp(); w != nil; w = resp() {
			write(getStatus(w), w)
		}
	default:
		write(graphql.StatusValidationError, graphql.ErrorResponse(ctx, "unsupported GraphQL operation"))
	}
}

func (e executor) parseOperation(ctx context.Context, rc *graphql.RequestContext) (*ast.QueryDocument, *gqlerror.Error) {
	ctx = rc.Tracer.StartOperationValidation(ctx)
	defer func() { rc.Tracer.EndOperationValidation(ctx) }()

	return parser.ParseQuery(&ast.Source{Input: rc.RawQuery})
}

func (e executor) validateOperation(ctx context.Context, rc *graphql.RequestContext) (context.Context, *ast.OperationDefinition, gqlerror.List) {
	ctx = rc.Tracer.StartOperationValidation(ctx)
	defer func() { rc.Tracer.EndOperationValidation(ctx) }()

	listErr := validator.Validate(e.es.Schema(), rc.Doc)
	if len(listErr) != 0 {
		return ctx, nil, listErr
	}

	op := rc.Doc.Operations.ForName(rc.OperationName)
	if op == nil {
		return ctx, nil, gqlerror.List{gqlerror.Errorf("operation %s not found", rc.OperationName)}
	}

	return ctx, op, nil
}

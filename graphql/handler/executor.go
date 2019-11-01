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
	operationMiddleware    graphql.OperationHandler
	responseMiddleware     graphql.ResponseMiddleware
	fieldMiddleware        graphql.FieldMiddleware
	requestParamMutators   []graphql.RequestParameterMutator
	requestContextMutators []graphql.RequestContextMutator
	server                 *Server
}

var _ graphql.GraphExecutor = executor{}

func newExecutor(s *Server) executor {
	e := executor{
		server: s,
	}
	e.operationMiddleware = e.executableSchemaHandler
	e.responseMiddleware = func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
		return next(ctx)
	}
	e.fieldMiddleware = func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		return next(ctx)
	}

	// this loop goes backwards so the first extension is the outer most middleware and runs first.
	for i := len(s.extensions) - 1; i >= 0; i-- {
		p := s.extensions[i]
		if p, ok := p.(graphql.OperationInterceptor); ok {
			previous := e.operationMiddleware
			e.operationMiddleware = func(ctx context.Context, writer graphql.Writer) {
				p.InterceptOperation(ctx, previous, writer)
			}
		}

		if p, ok := p.(graphql.ResponseInterceptor); ok {
			previous := e.responseMiddleware
			e.responseMiddleware = func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
				return p.InterceptResponse(ctx, func(ctx context.Context) *graphql.Response {
					return previous(ctx, next)
				})
			}
		}

		if p, ok := p.(graphql.FieldInterceptor); ok {
			previous := e.fieldMiddleware
			e.fieldMiddleware = func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
				return p.InterceptField(ctx, func(ctx context.Context) (res interface{}, err error) {
					return previous(ctx, next)
				})
			}
		}
	}

	for _, p := range s.extensions {
		if p, ok := p.(graphql.RequestParameterMutator); ok {
			e.requestParamMutators = append(e.requestParamMutators, p)
		}

		if p, ok := p.(graphql.RequestContextMutator); ok {
			e.requestContextMutators = append(e.requestContextMutators, p)
		}

	}

	return e
}

func (e executor) DispatchRequest(ctx context.Context, writer graphql.Writer) {
	e.operationMiddleware(ctx, writer)
}

func (e executor) CreateRequestContext(ctx context.Context, params *graphql.RawParams) (*graphql.RequestContext, gqlerror.List) {
	ctx = graphql.WithServerContext(ctx, e.server.es)

	stats := graphql.Stats{
		OperationStart: graphql.GetStartTime(ctx),
	}
	for _, p := range e.requestParamMutators {
		if err := p.MutateRequestParameters(ctx, params); err != nil {
			return nil, gqlerror.List{err}
		}
	}

	doc, listErr := e.parseQuery(ctx, &stats, params.Query)
	if len(listErr) != 0 {
		return nil, listErr
	}

	op := doc.Operations.ForName(params.OperationName)
	if op == nil {
		return nil, gqlerror.List{gqlerror.Errorf("operation %s not found", params.OperationName)}
	}

	vars, err := validator.VariableValues(e.server.es.Schema(), op, params.Variables)
	if err != nil {
		return nil, gqlerror.List{err}
	}
	stats.Validation.End = graphql.Now()

	rc := &graphql.RequestContext{
		RawQuery:             params.Query,
		Variables:            vars,
		OperationName:        params.OperationName,
		Doc:                  doc,
		Operation:            op,
		DisableIntrospection: true,
		Recover:              graphql.DefaultRecover,
		ResolverMiddleware:   e.fieldMiddleware,
		DirectiveMiddleware:  nil, //todo fixme
		Stats:                stats,
	}

	for _, p := range e.requestContextMutators {
		if err := p.MutateRequestContext(ctx, rc); err != nil {
			return nil, gqlerror.List{err}
		}
	}

	return rc, nil
}

func (e *executor) write(ctx context.Context, resp *graphql.Response, writer graphql.Writer) {
	resp.Extensions = graphql.GetExtensions(ctx)

	for _, err := range graphql.GetErrors(ctx) {
		resp.Errors = append(resp.Errors, err)
	}
	writer(getStatus(resp), resp)
}

// executableSchemaHandler is the inner most operation handler, it invokes the graph directly after all middleware
// and sends responses to the transport so it can be returned to the client
func (e *executor) executableSchemaHandler(ctx context.Context, write graphql.Writer) {
	responses := e.server.es.Exec(ctx)
	for {
		resCtx := graphql.WithResponseContext(ctx, e.server.errorPresenter, e.server.recoverFunc)
		resp := e.responseMiddleware(resCtx, func(ctx context.Context) *graphql.Response {
			resp := responses(resCtx)
			if resp == nil {
				return nil
			}
			resp.Extensions = graphql.GetExtensions(ctx)
			return resp
		})
		if resp == nil {
			break
		}
		e.write(resCtx, resp, write)
	}
}

// parseQuery decodes the incoming query and validates it, pulling from cache if present.
//
// NOTE: This should NOT look at variables, they will change per request. It should only parse and validate
// the raw query string.
func (e executor) parseQuery(ctx context.Context, stats *graphql.Stats, query string) (*ast.QueryDocument, gqlerror.List) {
	stats.Parsing.Start = graphql.Now()

	if doc, ok := e.server.queryCache.Get(query); ok {
		now := graphql.Now()

		stats.Parsing.End = now
		stats.Validation.Start = now
		return doc.(*ast.QueryDocument), nil
	}

	doc, err := parser.ParseQuery(&ast.Source{Input: query})
	if err != nil {
		return nil, gqlerror.List{err}
	}
	stats.Parsing.End = graphql.Now()

	stats.Validation.Start = graphql.Now()
	listErr := validator.Validate(e.server.es.Schema(), doc)
	if len(listErr) != 0 {
		return nil, listErr
	}

	e.server.queryCache.Add(query, doc)

	return doc, nil
}

package handler

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
)

type executor struct {
	operationMiddleware        graphql.OperationMiddleware
	responseMiddleware         graphql.ResponseMiddleware
	fieldMiddleware            graphql.FieldMiddleware
	operationParameterMutators []graphql.OperationParameterMutator
	operationContextMutators   []graphql.OperationContextMutator
	server                     *Server
}

var _ graphql.GraphExecutor = executor{}

func newExecutor(s *Server) executor {
	e := executor{
		server: s,
	}
	e.operationMiddleware = func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		return next(ctx)
	}
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
			e.operationMiddleware = func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
				return p.InterceptOperation(ctx, func(ctx context.Context) graphql.ResponseHandler {
					return previous(ctx, next)
				})
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
		if p, ok := p.(graphql.OperationParameterMutator); ok {
			e.operationParameterMutators = append(e.operationParameterMutators, p)
		}

		if p, ok := p.(graphql.OperationContextMutator); ok {
			e.operationContextMutators = append(e.operationContextMutators, p)
		}

	}

	return e
}

func (e executor) DispatchOperation(ctx context.Context, rc *graphql.OperationContext) (h graphql.ResponseHandler, resctx context.Context) {
	ctx = graphql.WithOperationContext(ctx, rc)

	var innerCtx context.Context
	res := e.operationMiddleware(ctx, func(ctx context.Context) graphql.ResponseHandler {
		innerCtx = ctx

		tmpResponseContext := graphql.WithResponseContext(ctx, e.server.errorPresenter, e.server.recoverFunc)
		responses := e.server.es.Exec(tmpResponseContext)
		if errs := graphql.GetErrors(tmpResponseContext); errs != nil {
			return graphql.OneShot(&graphql.Response{Errors: errs})
		}

		return func(ctx context.Context) *graphql.Response {
			ctx = graphql.WithResponseContext(ctx, e.server.errorPresenter, e.server.recoverFunc)
			resp := e.responseMiddleware(ctx, func(ctx context.Context) *graphql.Response {
				resp := responses(ctx)
				if resp == nil {
					return nil
				}
				resp.Errors = append(resp.Errors, graphql.GetErrors(ctx)...)
				resp.Extensions = graphql.GetExtensions(ctx)
				return resp
			})
			if resp == nil {
				return nil
			}

			return resp
		}
	})

	return res, innerCtx
}

func (e executor) CreateOperationContext(ctx context.Context, params *graphql.RawParams) (*graphql.OperationContext, gqlerror.List) {
	rc := &graphql.OperationContext{
		DisableIntrospection: true,
		Recover:              e.server.recoverFunc,
		ResolverMiddleware:   e.fieldMiddleware,
		Stats: graphql.Stats{
			Read:           params.ReadTime,
			OperationStart: graphql.GetStartTime(ctx),
		},
	}
	ctx = graphql.WithOperationContext(ctx, rc)

	for _, p := range e.operationParameterMutators {
		if err := p.MutateOperationParameters(ctx, params); err != nil {
			return rc, gqlerror.List{err}
		}
	}

	rc.RawQuery = params.Query
	rc.OperationName = params.OperationName

	var listErr gqlerror.List
	rc.Doc, listErr = e.parseQuery(ctx, &rc.Stats, params.Query)
	if len(listErr) != 0 {
		return rc, listErr
	}

	rc.Operation = rc.Doc.Operations.ForName(params.OperationName)
	if rc.Operation == nil {
		return rc, gqlerror.List{gqlerror.Errorf("operation %s not found", params.OperationName)}
	}

	var err *gqlerror.Error
	rc.Variables, err = validator.VariableValues(e.server.es.Schema(), rc.Operation, params.Variables)
	if err != nil {
		errcode.Set(err, errcode.ValidationFailed)
		return rc, gqlerror.List{err}
	}
	rc.Stats.Validation.End = graphql.Now()

	for _, p := range e.operationContextMutators {
		if err := p.MutateOperationContext(ctx, rc); err != nil {
			return rc, gqlerror.List{err}
		}
	}

	return rc, nil
}

func (e executor) DispatchError(ctx context.Context, list gqlerror.List) *graphql.Response {
	ctx = graphql.WithResponseContext(ctx, e.server.errorPresenter, e.server.recoverFunc)
	for _, gErr := range list {
		graphql.AddError(ctx, gErr)
	}

	resp := e.responseMiddleware(ctx, func(ctx context.Context) *graphql.Response {
		resp := &graphql.Response{
			Errors: list,
		}
		resp.Extensions = graphql.GetExtensions(ctx)
		return resp
	})

	return resp
}

// parseQuery decodes the incoming query and validates it, pulling from cache if present.
//
// NOTE: This should NOT look at variables, they will change per request. It should only parse and validate
// the raw query string.
func (e executor) parseQuery(ctx context.Context, stats *graphql.Stats, query string) (*ast.QueryDocument, gqlerror.List) {
	stats.Parsing.Start = graphql.Now()

	if doc, ok := e.server.queryCache.Get(ctx, query); ok {
		now := graphql.Now()

		stats.Parsing.End = now
		stats.Validation.Start = now
		return doc.(*ast.QueryDocument), nil
	}

	doc, err := parser.ParseQuery(&ast.Source{Input: query})
	if err != nil {
		errcode.Set(err, errcode.ParseFailed)
		return nil, gqlerror.List{err}
	}
	stats.Parsing.End = graphql.Now()

	stats.Validation.Start = graphql.Now()
	listErr := validator.Validate(e.server.es.Schema(), doc)
	if len(listErr) != 0 {
		for _, e := range listErr {
			errcode.Set(e, errcode.ValidationFailed)
		}
		return nil, listErr
	}

	e.server.queryCache.Add(ctx, query, doc)

	return doc, nil
}

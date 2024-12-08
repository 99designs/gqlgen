package executor

import (
	"context"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/vektah/gqlparser/v2/parser"
	"github.com/vektah/gqlparser/v2/validator"
	"github.com/vektah/gqlparser/v2/validator/rules"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
)

const parserTokenNoLimit = 0

// Executor executes graphql queries against a schema.
type Executor struct {
	es         graphql.ExecutableSchema
	extensions []graphql.HandlerExtension
	ext        extensions

	errorPresenter graphql.ErrorPresenterFunc
	recoverFunc    graphql.RecoverFunc
	queryCache     graphql.Cache[*ast.QueryDocument]

	parserTokenLimit  int
	disableSuggestion bool
}

var _ graphql.GraphExecutor = &Executor{}

// New creates a new Executor with the given schema, and a default error and
// recovery callbacks, and no query cache or extensions.
func New(es graphql.ExecutableSchema) *Executor {
	e := &Executor{
		es:               es,
		errorPresenter:   graphql.DefaultErrorPresenter,
		recoverFunc:      graphql.DefaultRecover,
		queryCache:       graphql.NoCache[*ast.QueryDocument]{},
		ext:              processExtensions(nil),
		parserTokenLimit: parserTokenNoLimit,
	}
	return e
}

func (e *Executor) CreateOperationContext(
	ctx context.Context,
	params *graphql.RawParams,
) (*graphql.OperationContext, gqlerror.List) {
	opCtx := &graphql.OperationContext{
		DisableIntrospection:   true,
		RecoverFunc:            e.recoverFunc,
		ResolverMiddleware:     e.ext.fieldMiddleware,
		RootResolverMiddleware: e.ext.rootFieldMiddleware,
		Stats: graphql.Stats{
			Read:           params.ReadTime,
			OperationStart: graphql.GetStartTime(ctx),
		},
	}
	ctx = graphql.WithOperationContext(ctx, opCtx)

	for _, p := range e.ext.operationParameterMutators {
		if err := p.MutateOperationParameters(ctx, params); err != nil {
			return opCtx, gqlerror.List{err}
		}
	}

	opCtx.RawQuery = params.Query
	opCtx.OperationName = params.OperationName
	opCtx.Headers = params.Headers

	var listErr gqlerror.List
	opCtx.Doc, listErr = e.parseQuery(ctx, &opCtx.Stats, params.Query)
	if len(listErr) != 0 {
		return opCtx, listErr
	}

	opCtx.Operation = opCtx.Doc.Operations.ForName(params.OperationName)
	if opCtx.Operation == nil {
		err := gqlerror.Errorf("operation %s not found", params.OperationName)
		errcode.Set(err, errcode.ValidationFailed)
		return opCtx, gqlerror.List{err}
	}

	var err error
	opCtx.Variables, err = validator.VariableValues(e.es.Schema(), opCtx.Operation, params.Variables)
	if err != nil {
		gqlErr, ok := err.(*gqlerror.Error)
		if ok {
			errcode.Set(gqlErr, errcode.ValidationFailed)
			return opCtx, gqlerror.List{gqlErr}
		}
	}
	opCtx.Stats.Validation.End = graphql.Now()

	for _, p := range e.ext.operationContextMutators {
		if err := p.MutateOperationContext(ctx, opCtx); err != nil {
			return opCtx, gqlerror.List{err}
		}
	}

	return opCtx, nil
}

func (e *Executor) DispatchOperation(
	ctx context.Context,
	opCtx *graphql.OperationContext,
) (graphql.ResponseHandler, context.Context) {
	ctx = graphql.WithOperationContext(ctx, opCtx)

	var innerCtx context.Context
	res := e.ext.operationMiddleware(ctx, func(ctx context.Context) graphql.ResponseHandler {
		innerCtx = ctx

		tmpResponseContext := graphql.WithResponseContext(ctx, e.errorPresenter, e.recoverFunc)
		responses := e.es.Exec(tmpResponseContext)
		if errs := graphql.GetErrors(tmpResponseContext); errs != nil {
			return graphql.OneShot(&graphql.Response{Errors: errs})
		}

		return func(ctx context.Context) *graphql.Response {
			ctx = graphql.WithResponseContext(ctx, e.errorPresenter, e.recoverFunc)
			resp := e.ext.responseMiddleware(ctx, func(ctx context.Context) *graphql.Response {
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

func (e *Executor) DispatchError(ctx context.Context, list gqlerror.List) *graphql.Response {
	ctx = graphql.WithResponseContext(ctx, e.errorPresenter, e.recoverFunc)
	for _, gErr := range list {
		graphql.AddError(ctx, gErr)
	}

	resp := e.ext.responseMiddleware(ctx, func(ctx context.Context) *graphql.Response {
		resp := &graphql.Response{
			Errors: graphql.GetErrors(ctx),
		}
		resp.Extensions = graphql.GetExtensions(ctx)
		return resp
	})

	return resp
}

func (e *Executor) PresentRecoveredError(ctx context.Context, err any) error {
	return e.errorPresenter(ctx, e.recoverFunc(ctx, err))
}

func (e *Executor) SetQueryCache(cache graphql.Cache[*ast.QueryDocument]) {
	e.queryCache = cache
}

func (e *Executor) SetErrorPresenter(f graphql.ErrorPresenterFunc) {
	e.errorPresenter = f
}

func (e *Executor) SetRecoverFunc(f graphql.RecoverFunc) {
	e.recoverFunc = f
}

func (e *Executor) SetParserTokenLimit(limit int) {
	e.parserTokenLimit = limit
}

func (e *Executor) SetDisableSuggestion(value bool) {
	e.disableSuggestion = value
}

// parseQuery decodes the incoming query and validates it, pulling from cache if present.
//
// NOTE: This should NOT look at variables, they will change per request. It should only parse and
// validate
// the raw query string.
func (e *Executor) parseQuery(
	ctx context.Context,
	stats *graphql.Stats,
	query string,
) (*ast.QueryDocument, gqlerror.List) {
	stats.Parsing.Start = graphql.Now()

	if doc, ok := e.queryCache.Get(ctx, query); ok {
		now := graphql.Now()

		stats.Parsing.End = now
		stats.Validation.Start = now
		return doc, nil
	}

	doc, err := parser.ParseQueryWithTokenLimit(&ast.Source{Input: query}, e.parserTokenLimit)
	if err != nil {
		gqlErr, ok := err.(*gqlerror.Error)
		if ok {
			errcode.Set(gqlErr, errcode.ParseFailed)
			return nil, gqlerror.List{gqlErr}
		}
	}
	stats.Parsing.End = graphql.Now()

	stats.Validation.Start = graphql.Now()

	if len(doc.Operations) == 0 {
		err = gqlerror.Errorf("no operation provided")
		gqlErr, _ := err.(*gqlerror.Error)
		errcode.Set(err, errcode.ValidationFailed)
		return nil, gqlerror.List{gqlErr}
	}

	// swap out the FieldsOnCorrectType rule with one that doesn't provide suggestions
	if e.disableSuggestion {
		validator.RemoveRule("FieldsOnCorrectType")

		rule := rules.FieldsOnCorrectTypeRuleWithoutSuggestions
		validator.AddRule(rule.Name, rule.RuleFunc)
	}

	listErr := validator.Validate(e.es.Schema(), doc)
	if len(listErr) != 0 {
		for _, e := range listErr {
			errcode.Set(e, errcode.ValidationFailed)
		}
		return nil, listErr
	}

	e.queryCache.Add(ctx, query, doc)

	return doc, nil
}

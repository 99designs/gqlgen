package handler

import "github.com/99designs/gqlgen/graphql"

func newRequestContext() *graphql.RequestContext {
	return &graphql.RequestContext{
		DisableIntrospection: true,
		Recover:              graphql.DefaultRecover,
		ErrorPresenter:       graphql.DefaultErrorPresenter,
		ResolverMiddleware:   nil,
		RequestMiddleware:    nil,
		Tracer:               graphql.NopTracer{},
		ComplexityLimit:      0,
	}
}

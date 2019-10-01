package handler

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
)

type middlewareContext struct {
	*graphql.RequestContext
	InvokedNext bool
}

func testMiddleware(m Middleware, initialContexts ...graphql.RequestContext) middlewareContext {
	rc := &graphql.RequestContext{}
	if len(initialContexts) > 0 {
		rc = &initialContexts[0]
	}

	m(func(ctx context.Context, writer Writer) {
		rc = graphql.GetRequestContext(ctx)
	})(graphql.WithRequestContext(context.Background(), rc), noopWriter)

	return middlewareContext{
		InvokedNext:    rc != nil,
		RequestContext: rc,
	}
}

func noopWriter(response *graphql.Response) {}

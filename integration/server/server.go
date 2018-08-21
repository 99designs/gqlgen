package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
	"github.com/99designs/gqlgen/integration"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/gqlerror"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	http.Handle("/", handler.Playground("GraphQL playground", "/query"))
	http.Handle("/query", handler.GraphQL(
		integration.NewExecutableSchema(integration.Config{Resolvers: &integration.Resolver{}}),
		handler.ErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
			if e, ok := errors.Cause(e).(*integration.CustomError); ok {
				return &gqlerror.Error{
					Message: e.UserMessage,
					Path:    graphql.GetResolverContext(ctx).Path(),
				}
			}
			return graphql.DefaultErrorPresenter(ctx, e)
		}),
	))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/playground"

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

	cfg := integration.Config{Resolvers: &integration.Resolver{}}
	cfg.Complexity.Query.Complexity = func(childComplexity, value int) int {
		// Allow the integration client to dictate the complexity, to verify this
		// function is executed.
		return value
	}
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", handler.GraphQL(
		integration.NewExecutableSchema(cfg),
		handler.ErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
			if e, ok := errors.Cause(e).(*integration.CustomError); ok {
				return &gqlerror.Error{
					Message: e.UserMessage,
					Path:    graphql.GetFieldContext(ctx).Path(),
				}
			}
			return graphql.DefaultErrorPresenter(ctx, e)
		}),
		handler.ComplexityLimit(1000),
	))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

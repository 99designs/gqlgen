package main

import (
	"context"
	"github.com/99designs/gqlgen/example/cache/graph/generated"
	"github.com/99designs/gqlgen/example/cache/graph/model"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/example/cache/graph"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	cfg := generated.Config{Resolvers: &graph.Resolver{}}
	cfg.Directives.CacheControl = func(ctx context.Context, obj interface{}, next graphql.Resolver, maxAge *int, scope *model.CacheControlScope) (res interface{}, err error) {
		res, err = next(ctx)
		if err == nil && maxAge != nil {
			graphql.SetCacheHint(ctx, "PUBLIC", time.Duration(*maxAge)*time.Second)
		}

		return
	}

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(cfg))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}






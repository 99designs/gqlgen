package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/99designs/gqlgen/integration/server"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	cfg := server.Config{Resolvers: &server.Resolver{}}
	cfg.Complexity.Query.Complexity = func(childComplexity, value int) int {
		// Allow the integration client to dictate the complexity, to verify this
		// function is executed.
		return value
	}

	srv := handler.New(server.NewExecutableSchema(cfg))

	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.SSE{})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New(1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	srv.SetErrorPresenter(func(ctx context.Context, e error) *gqlerror.Error {
		var ie *server.CustomError
		if errors.As(e, &ie) {
			return &gqlerror.Error{
				Message: ie.UserMessage,
				Path:    graphql.GetPath(ctx),
			}
		}
		return graphql.DefaultErrorPresenter(ctx, e)
	})
	srv.Use(extension.FixedComplexityLimit(1000))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

package main

import (
	"log"
	"net/http"
	"os"

	extension "github.com/john-markham/gqlgen/_examples/type-system-extension"
	"github.com/john-markham/gqlgen/graphql/handler"
	"github.com/john-markham/gqlgen/graphql/handler/transport"
	"github.com/john-markham/gqlgen/graphql/playground"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := handler.New(
		extension.NewExecutableSchema(
			extension.Config{
				Resolvers: extension.NewRootResolver(),
				Directives: extension.DirectiveRoot{
					EnumLogging:   extension.EnumLogging,
					FieldLogging:  extension.FieldLogging,
					InputLogging:  extension.InputLogging,
					ObjectLogging: extension.ObjectLogging,
					ScalarLogging: extension.ScalarLogging,
					UnionLogging:  extension.UnionLogging,
				},
			},
		),
	)
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

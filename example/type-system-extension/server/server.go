package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/example/type-system-extension"
	"github.com/99designs/gqlgen/handler"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	http.Handle("/", handler.Playground("GraphQL playground", "/query"))
	http.Handle("/query", handler.GraphQL(
		type_system_extension.NewExecutableSchema(
			type_system_extension.Config{
				Resolvers: type_system_extension.NewRootResolver(),
				Directives: type_system_extension.DirectiveRoot{
					EnumLogging:   type_system_extension.EnumLogging,
					FieldLogging:  type_system_extension.FieldLogging,
					InputLogging:  type_system_extension.InputLogging,
					ObjectLogging: type_system_extension.ObjectLogging,
					ScalarLogging: type_system_extension.ScalarLogging,
					UnionLogging:  type_system_extension.UnionLogging,
				},
			},
		),
	))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

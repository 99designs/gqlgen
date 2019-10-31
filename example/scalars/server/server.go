package main

import (
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/99designs/gqlgen/example/scalars"
	"github.com/99designs/gqlgen/handler"
)

func main() {
	http.Handle("/", playground.Handler("Starwars", "/query"))
	http.Handle("/query", handler.GraphQL(scalars.NewExecutableSchema(scalars.Config{Resolvers: &scalars.Resolver{}})))

	log.Fatal(http.ListenAndServe(":8084", nil))
}

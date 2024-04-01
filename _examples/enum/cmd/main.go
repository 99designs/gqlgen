package main

import (
	"log"
	"net/http"

	"github.com/99designs/gqlgen/_examples/enum/api"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	http.Handle("/", playground.Handler("Enum", "/query"))
	http.Handle("/query", handler.NewDefaultServer(api.NewExecutableSchema(api.Config{Resolvers: &api.Resolver{}})))
	log.Fatal(http.ListenAndServe(":8081", nil))
}

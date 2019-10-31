package main

import (
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/playground"

	todo "github.com/99designs/gqlgen/example/config"
	"github.com/99designs/gqlgen/handler"
)

func main() {
	http.Handle("/", playground.Handler("Todo", "/query"))
	http.Handle("/query", handler.GraphQL(
		todo.NewExecutableSchema(todo.New()),
	))
	log.Fatal(http.ListenAndServe(":8081", nil))
}

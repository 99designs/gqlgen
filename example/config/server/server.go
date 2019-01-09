package main

import (
	"log"
	"net/http"

	todo "github.com/monzo/gqlgen/example/config"
	"github.com/monzo/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("Todo", "/query"))
	http.Handle("/query", handler.GraphQL(
		todo.NewExecutableSchema(todo.New()),
	))
	log.Fatal(http.ListenAndServe(":8081", nil))
}

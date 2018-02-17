package main

import (
	"log"
	"net/http"

	"github.com/vektah/gqlgen/example/todo"
	"github.com/vektah/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("Todo", "/query"))
	http.Handle("/query", handler.GraphQL(todo.MakeExecutableSchema(todo.New())))
	log.Fatal(http.ListenAndServe(":8081", nil))
}

package main

import (
	"log"
	"net/http"

	"github.com/vektah/gqlgen/example/todo"
	"github.com/vektah/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.GraphiQL("Todo", "/query"))
	http.Handle("/query", handler.GraphQL(todo.NewExecutor(todo.New())))
	log.Fatal(http.ListenAndServe(":8081", nil))
}

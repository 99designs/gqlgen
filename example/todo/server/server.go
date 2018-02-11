package main

import (
	"log"
	"net/http"

	graphql "github.com/vektah/graphql-go"
	"github.com/vektah/graphql-go/example/todo"
)

func main() {
	http.Handle("/", graphql.GraphiqlHandler("Todo", "/query"))
	http.Handle("/query", graphql.Handler(todo.NewExecutor(todo.New())))
	log.Fatal(http.ListenAndServe(":8081", nil))
}

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/99designs/gqlgen/example/todo"
	"github.com/99designs/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("Todo", "/query"))
	http.Handle("/query", handler.GraphQL(
		todo.NewExecutableSchema(todo.New()),
		handler.RecoverFunc(func(ctx context.Context, err interface{}) error {
			// send this panic somewhere
			log.Print(err)
			debug.PrintStack()
			return errors.New("user message on panic")
		}),
	))
	log.Fatal(http.ListenAndServe(":8081", nil))
}

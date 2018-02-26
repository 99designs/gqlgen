package main

import (
	"errors"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/vektah/gqlgen/example/todo"
	"github.com/vektah/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("Todo", "/query"))
	http.Handle("/query", handler.GraphQL(
		todo.MakeExecutableSchema(todo.New()),
		handler.RecoverFunc(func(err interface{}) error {
			log.Printf("send this panic somewhere")
			debug.PrintStack()
			return errors.New("user message on panic")
		}),
	))
	log.Fatal(http.ListenAndServe(":8081", nil))
}

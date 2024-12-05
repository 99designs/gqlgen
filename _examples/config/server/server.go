package main

import (
	"log"
	"net/http"

	todo "github.com/99designs/gqlgen/_examples/config"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	srv := handler.New(
		todo.NewExecutableSchema(todo.New()),
	)
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	http.Handle("/", playground.Handler("Todo", "/query"))
	http.Handle("/query", srv)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

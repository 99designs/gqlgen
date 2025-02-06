package main

import (
	"log"
	"net/http"

	"github.com/99designs/gqlgen/_examples/selection"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	srv := handler.New(
		selection.NewExecutableSchema(selection.Config{Resolvers: &selection.Resolver{}}),
	)
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	http.Handle("/", playground.Handler("Selection Demo", "/query"))
	http.Handle("/query", srv)
	log.Fatal(http.ListenAndServe(":8086", nil))
}

package main

import (
	"log"
	"net/http"

	"github.com/99designs/gqlgen/_examples/enum/api"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	srv := handler.New(
		api.NewExecutableSchema(api.Config{Resolvers: &api.Resolver{}}),
	)

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	http.Handle("/", playground.Handler("Enum", "/query"))
	http.Handle("/query", srv)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

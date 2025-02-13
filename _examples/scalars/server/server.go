package main

import (
	"log"
	"net/http"

	"github.com/john-markham/gqlgen/_examples/scalars"
	"github.com/john-markham/gqlgen/graphql/handler"
	"github.com/john-markham/gqlgen/graphql/handler/transport"
	"github.com/john-markham/gqlgen/graphql/playground"
)

func main() {
	srv := handler.New(
		scalars.NewExecutableSchema(scalars.Config{Resolvers: &scalars.Resolver{}}),
	)
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	http.Handle("/", playground.Handler("Starwars", "/query"))
	http.Handle("/query", srv)

	log.Fatal(http.ListenAndServe(":8084", nil))
}

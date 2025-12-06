package main

import (
	"log"
	"net/http"

	unionextension "github.com/99designs/gqlgen/_examples/union-extension"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	srv := handler.New(
		unionextension.NewExecutableSchema(
			unionextension.Config{Resolvers: &unionextension.Resolver{}},
		),
	)
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	http.Handle("/", playground.Handler("Union Extension Demo", "/query"))
	http.Handle("/query", srv)
	log.Fatal(http.ListenAndServe(":8086", nil))
}

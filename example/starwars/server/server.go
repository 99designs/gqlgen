package main

import (
	"log"
	"net/http"

	"github.com/vektah/gqlgen/example/starwars"
	"github.com/vektah/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("Starwars", "/query"))
	http.Handle("/query", handler.GraphQL(starwars.MakeExecutableSchema(starwars.NewResolver())))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

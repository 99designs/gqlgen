package main

import (
	"log"
	"net/http"

	"github.com/vektah/gqlgen/example/starwars"
	"github.com/vektah/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.GraphiQL("Starwars", "/query"))
	http.Handle("/query", handler.GraphQL(starwars.NewExecutor(starwars.NewResolver())))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

package main

import (
	"log"
	"net/http"

	"github.com/vektah/graphql-go"
	"github.com/vektah/graphql-go/example/starwars"
)

func main() {
	http.Handle("/", graphql.GraphiqlHandler("Starwars", "/query"))
	http.Handle("/query", graphql.Handler(starwars.NewExecutor(starwars.NewResolver())))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

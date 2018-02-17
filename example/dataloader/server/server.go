package main

import (
	"log"
	"net/http"

	"github.com/vektah/gqlgen/example/dataloader"
	"github.com/vektah/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("Dataloader", "/query"))

	http.Handle("/query", dataloader.LoaderMiddleware(handler.GraphQL(dataloader.MakeExecutableSchema(&dataloader.Resolver{}))))

	log.Fatal(http.ListenAndServe(":8082", nil))
}

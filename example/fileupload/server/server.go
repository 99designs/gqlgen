package main

import (
	"log"
	"net/http"

	"github.com/99designs/gqlgen/example/fileupload"
	"github.com/99designs/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("File Upload Demo", "/query"))
	http.Handle("/query", handler.GraphQL(fileupload.NewExecutableSchema(fileupload.Config{Resolvers: &fileupload.Resolver{}})))
	log.Fatal(http.ListenAndServe(":8086", nil))
}

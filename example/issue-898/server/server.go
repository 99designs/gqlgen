package main

import (
	"log"
	"net/http"
	"os"

	issue898 "github.com/99designs/gqlgen/example/issue-898"
	"github.com/99designs/gqlgen/handler"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	http.Handle("/", handler.Playground("GraphQL playground", "/query"))
	http.Handle("/query", handler.GraphQL(issue898.NewExecutableSchema(issue898.Config{Resolvers: &issue898.Resolver{}})))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

package main

import (
	log "log"
	http "net/http"
	os "os"

	gqlapollotracingtest "github.com/99designs/gqlgen/gqlapollotracing/internal/gqlapollotracingtest"
	handler "github.com/99designs/gqlgen/handler"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	http.Handle("/", handler.Playground("GraphQL playground", "/query"))
	http.Handle("/query", handler.GraphQL(gqlapollotracingtest.NewExecutableSchema(gqlapollotracingtest.Config{Resolvers: &gqlapollotracingtest.Resolver{}})))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

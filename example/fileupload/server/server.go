package main

import (
	"context"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/example/fileupload"
	"github.com/99designs/gqlgen/example/fileupload/model"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("File Upload Demo", "/query"))
	resolver := &fileupload.Resolver{
		SingleUploadFunc: func(ctx context.Context, file graphql.Upload) (*model.File, error) {
			return &model.File{
				ID: 1,
			},nil
		},
	}
	http.Handle("/query", handler.GraphQL(fileupload.NewExecutableSchema(fileupload.Config{Resolvers: resolver})))

	log.Print("connect to http://localhost:8087/ for GraphQL playground")
	log.Fatal(http.ListenAndServe(":8087", nil))
}

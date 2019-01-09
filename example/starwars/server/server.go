package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/monzo/gqlgen/example/starwars"
	"github.com/monzo/gqlgen/graphql"
	"github.com/monzo/gqlgen/handler"
)

func main() {
	http.Handle("/", handler.Playground("Starwars", "/query"))
	http.Handle("/query", handler.GraphQL(starwars.NewExecutableSchema(starwars.NewResolver()),
		handler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
			rc := graphql.GetResolverContext(ctx)
			fmt.Println("Entered", rc.Object, rc.Field.Name)
			res, err = next(ctx)
			fmt.Println("Left", rc.Object, rc.Field.Name, "=>", res, err)
			return res, err
		}),
	))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

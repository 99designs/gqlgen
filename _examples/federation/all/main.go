package main

import (
	"context"
	"log"

	accounts "github.com/99designs/gqlgen/_examples/federation/accounts/schema"
	products "github.com/99designs/gqlgen/_examples/federation/products/schema"
	reviews "github.com/99designs/gqlgen/_examples/federation/reviews/schema"
	"github.com/99designs/gqlgen/_examples/federation/subgraphs"
)

func main() {
	ctx := context.Background()
	subgraphs, err := subgraphs.New(
		ctx,
		subgraphs.SubgraphConfig{
			Name:   "accounts",
			Schema: accounts.Schema,
			Port:   accounts.DefaultPort,
		},
		subgraphs.SubgraphConfig{
			Name:   "products",
			Schema: products.Schema,
			Port:   products.DefaultPort,
		},
		subgraphs.SubgraphConfig{
			Name:   "reviews",
			Schema: reviews.Schema,
			Port:   reviews.DefaultPort,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(subgraphs.ListenAndServe(ctx))
}

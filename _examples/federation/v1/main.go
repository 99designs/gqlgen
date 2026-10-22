// Command v1 serves the Federation v1 subgraphs that the integration test's
// second gateway composes.
package main

import (
	"context"
	"log"

	"github.com/99designs/gqlgen/_examples/federation/subgraphs"
	accounts "github.com/99designs/gqlgen/_examples/federation/v1/accounts/schema"
	content "github.com/99designs/gqlgen/_examples/federation/v1/content/schema"
	progress "github.com/99designs/gqlgen/_examples/federation/v1/progress/schema"
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
			Name:   "content",
			Schema: content.Schema,
			Port:   content.DefaultPort,
		},
		subgraphs.SubgraphConfig{
			Name:   "progress",
			Schema: progress.Schema,
			Port:   progress.DefaultPort,
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(subgraphs.ListenAndServe(ctx))
}

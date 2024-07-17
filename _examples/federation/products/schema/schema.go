package schema

import (
	"github.com/99designs/gqlgen/_examples/federation/products/graph"
)

const DefaultPort = "4002"

var Schema = graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}})

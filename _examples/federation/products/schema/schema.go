package schema

import (
	"github.com/john-markham/gqlgen/_examples/federation/products/graph"
)

const DefaultPort = "4002"

var Schema = graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}})

package schema

import (
	"github.com/john-markham/gqlgen/_examples/federation/reviews/graph"
)

const DefaultPort = "4003"

var Schema = graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}})

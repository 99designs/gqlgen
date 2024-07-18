package schema

import (
	"github.com/99designs/gqlgen/_examples/federation/reviews/graph"
)

const DefaultPort = "4003"

var Schema = graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}})

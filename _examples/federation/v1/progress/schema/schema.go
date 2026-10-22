//go:generate go run ../../../../../testdata/gqlgen.go

package schema

import (
	"github.com/99designs/gqlgen/_examples/federation/v1/progress/graph"
)

const DefaultPort = "4013"

var Schema = graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}})

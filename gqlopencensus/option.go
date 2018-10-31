package gqlopencensus

import "github.com/99designs/gqlgen/graphql"

type config struct {
	tracer graphql.Tracer
}

// Option is anything that can configure Tracer.
type Option func(cfg *config)

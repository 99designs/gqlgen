package singlefile

import (
	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func newDefaultClient(schema graphql.ExecutableSchema) *client.Client {
	srv := handler.New(schema)
	srv.AddTransport(transport.POST{})
	return client.New(srv)
}

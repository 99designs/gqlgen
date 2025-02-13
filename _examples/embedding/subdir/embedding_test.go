package subdir

import (
	"testing"

	"github.com/john-markham/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"

	"github.com/john-markham/gqlgen/_examples/embedding/subdir/gendir"
	"github.com/john-markham/gqlgen/client"
	"github.com/john-markham/gqlgen/graphql/handler"
)

func TestEmbeddingWorks(t *testing.T) {
	srv := handler.New(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)
	var resp struct {
		InSchemadir string
		Parentdir   string
		Subdir      string
	}
	c.MustPost(`{
				inSchemadir
				parentdir
				subdir
			}
		`, &resp)

	require.Equal(t, "example", resp.InSchemadir)
	require.Equal(t, "example", resp.Parentdir)
	require.Equal(t, "example", resp.Subdir)
}

func TestEmbeddingWorksInGendir(t *testing.T) {
	srv := handler.New(gendir.NewExecutableSchema(gendir.Config{Resolvers: &GendirResolver{}}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)
	var resp struct {
		InSchemadir string
		Parentdir   string
		Subdir      string
	}
	c.MustPost(`{
				inSchemadir
				parentdir
				subdir
			}
		`, &resp)

	require.Equal(t, "example", resp.InSchemadir)
	require.Equal(t, "example", resp.Parentdir)
	require.Equal(t, "example", resp.Subdir)
}

//go:generate rm -rf internal/gqlgenexec
//go:generate rm -f resolver.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package splitpackages

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
)

func TestSplitPackagesLayout(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Hello = func(ctx context.Context, name string) (string, error) {
		return "Hello " + name, nil
	}

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{
		Resolvers: resolvers,
	}))
	c := client.New(srv)

	var resp struct {
		Hello string
	}
	c.MustPost(`query { hello(name:"Ada") }`, &resp)
	require.Equal(t, "Hello Ada", resp.Hello)
}

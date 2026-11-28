package returnpointersinunmarshalinput

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// TestUnmarshalNamedInputWithPointers unmarshals inputs by name from a resolver, through
// the index of the unmarshalers that return pointers.
func TestUnmarshalNamedInputWithPointers(t *testing.T) {
	raw := map[string]any{}
	resolvers := &Stub{}
	resolvers.QueryResolver.Search = func(ctx context.Context, _ SearchFilters) (string, error) {
		var ptr *SearchFilters
		if err := graphql.UnmarshalNamedInputFromContext(
			ctx,
			"SearchFilters",
			raw,
			&ptr,
		); err != nil {
			return "", err
		}
		var value SearchFilters
		if err := graphql.UnmarshalNamedInputFromContext(
			ctx,
			"SearchFilters",
			raw,
			&value,
		); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %d / %s %d", *ptr.Name, *ptr.Limit, *value.Name, *value.Limit), nil
	}
	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	query := func() (string, error) {
		var resp struct{ Search string }
		err := c.Post(`{ search(filters: {}) }`, &resp)
		return resp.Search, err
	}

	raw = map[string]any{"name": "bob", "limit": 7}
	got, err := query()
	require.NoError(t, err)
	require.Equal(t, "bob 7 / bob 7", got)

	raw = map[string]any{"name": "bob", "limit": "seven"}
	_, err = query()
	require.Equal(
		t,
		[]string{`strconv.ParseInt: parsing "seven": invalid syntax`},
		messages(t, err),
	)
}

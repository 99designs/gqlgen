// Package modes compares the executors that both exec modes generate for the schemas of
// the tablemode test server whose table-mode package may fail when it is initialized,
// which the tests of the tablemode package cannot import without failing all of them.
package modes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// Compare requires the responses of both executable schemas to each query to be the
// same, and to contain contains.
func Compare(
	t *testing.T,
	functions, table func() graphql.ExecutableSchema,
	query, contains string,
) {
	t.Helper()
	var want, got string
	require.NotPanics(t, func() { want = respond(t, functions(), query) })
	require.NotPanics(t, func() { got = respond(t, table(), query) })
	require.Equal(t, want, got)
	require.Contains(t, want, contains)
}

// respond returns the response of es to query, its data and its errors, as JSON.
func respond(t *testing.T, es graphql.ExecutableSchema, query string) string {
	t.Helper()
	h := handler.New(es)
	h.AddTransport(transport.POST{})
	resp, err := client.New(h).RawPost(query)
	require.NoError(t, err)
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	return string(b)
}

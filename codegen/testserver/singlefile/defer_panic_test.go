package singlefile

import (
	"context"
	"encoding/json"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// A deferred field whose resolver panics must fail like one whose resolver
// returned an error: the initial response stands, the incremental payload
// reports the field as null alongside an error, and the delivery ends. The
// value may not vanish from the payload.
func TestDeferPanickingResolver(t *testing.T) {
	t.Parallel()

	resolvers := &Stub{}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.SSE{})

	c := client.New(srv)

	resolvers.QueryResolver.DeferSingle = func(ctx context.Context) (*DeferModel, error) {
		return &DeferModel{
			ID:   "1",
			Name: "Defer test 1",
		}, nil
	}
	resolvers.DeferModelResolver.Values = func(ctx context.Context, obj *DeferModel) ([]string, error) {
		panic("boom")
	}

	type (
		deferPanicInitial struct {
			Data struct {
				DeferSingle struct {
					ID     string   `json:"id"`
					Name   string   `json:"name"`
					Values []string `json:"values"`
				} `json:"deferSingle"`
			} `json:"data"`
			Label      string          `json:"label"`
			Path       []any           `json:"path"`
			Errors     json.RawMessage `json:"errors"`
			HasNext    bool            `json:"hasNext"`
			Extensions map[string]any  `json:"extensions"`
		}

		deferPanicPayload struct {
			Data       any             `json:"data"`
			Label      string          `json:"label"`
			Path       []any           `json:"path"`
			Errors     json.RawMessage `json:"errors"`
			HasNext    bool            `json:"hasNext"`
			Extensions map[string]any  `json:"extensions"`
		}
	)

	query := `query testDeferPanickingResolver {
	deferSingle {
		id
		name
		... @defer {
			values
		}
	}
}`

	synctest.Test(t, func(t *testing.T) {
		read := c.SSE(context.Background(), query)

		var initial deferPanicInitial
		synctest.Wait()
		require.NoError(t, read.Next(&initial), "expected an initial response")
		require.Equal(t, "1", initial.Data.DeferSingle.ID)
		assert.Equal(t, "Defer test 1", initial.Data.DeferSingle.Name)
		assert.Empty(t, initial.Data.DeferSingle.Values,
			"the initial response reports a deferred field as null")
		assert.Empty(t, initial.Errors, "the panic has not happened yet")
		assert.True(t, initial.HasNext, "the deferred field keeps the response open")

		var payload deferPanicPayload
		synctest.Wait()
		var rawErr client.RawJsonError
		require.ErrorAs(t, read.Next(&payload), &rawErr,
			"the payload for the panicking defer label reports the recovered panic")
		assert.Contains(t, string(rawErr.RawMessage), "internal system error")
		assert.Nil(t, payload.Data,
			"a panicked non-null deferred field nulls its payload instead of vanishing from it")
		assert.Equal(t, []any{"deferSingle"}, payload.Path)
		assert.False(t, payload.HasNext, "the panicking label is the last payload")

		require.NoError(t, read.Close(), "expected to close reader")
	})
}

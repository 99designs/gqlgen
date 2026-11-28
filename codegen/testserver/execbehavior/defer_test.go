package execbehavior

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefer(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.DeferItem = func(ctx context.Context) (*DeferItem, error) {
		return &DeferItem{ID: 1}, nil
	}
	resolvers.QueryResolver.DeferItems = func(ctx context.Context) ([]*DeferItem, error) {
		return []*DeferItem{{ID: 1}, {ID: 2}}, nil
	}
	resolvers.DeferItemResolver.Slow = func(ctx context.Context, obj *DeferItem) (string, error) {
		return "slow", nil
	}
	resolvers.DeferItemResolver.Failing = func(ctx context.Context, obj *DeferItem) (*string, error) {
		return nil, errors.New("failing field failed")
	}
	resolvers.DeferItemResolver.NonNullFailing = func(ctx context.Context, obj *DeferItem) (string, error) {
		return "", errors.New("non-null field failed")
	}
	es := NewExecutableSchema(Config{Resolvers: resolvers})

	t.Run("without defer", func(t *testing.T) {
		got := execute(t, es, `{ deferItem { id slow } }`)
		require.JSONEq(t, `[{"data":{"deferItem":{"id":1,"slow":"slow"}}}]`, got)
	})

	// KNOWN BUG: a deferred field is sent as null in the first response, and with its
	// value in the response of its label.
	// Expected: the first response leaves the deferred field out, because the @defer
	// proposal sends a deferred field only with its label. slow is non-null, so the
	// null breaks the schema, and a client cannot tell it from a resolved null.
	// Issue: https://github.com/99designs/gqlgen/issues/4458
	t.Run("deferred field", func(t *testing.T) {
		got := execute(t, es, `{ deferItem { id ... @defer(label: "later") { slow } } }`)
		require.JSONEq(t, `[
			{"data":{"deferItem":{"id":1,"slow":null}},"hasNext":true},
			{"data":{"slow":"slow"},"label":"later","path":["deferItem"]}
		]`, got)
	})

	t.Run("defer with if false", func(t *testing.T) {
		got := execute(t, es, `{ deferItem { id ... @defer(label: "later", if: false) { slow } } }`)
		require.JSONEq(t, `[{"data":{"deferItem":{"id":1,"slow":"slow"}}}]`, got)
	})

	t.Run("fields that are not resolved concurrently are not deferred", func(t *testing.T) {
		got := execute(t, es, `{ deferItem { ... @defer(label: "later") { id } slow } }`)
		require.JSONEq(t, `[{"data":{"deferItem":{"id":1,"slow":"slow"}}}]`, got)
	})

	// KNOWN BUG: the errors of the deferred fields of one object are collected together,
	// and each deferred response takes those collected when its label completes, so an
	// error goes with the first label to complete after it, not with its own.
	// Expected: the error is sent with label b, because the errors of a field go with
	// the response that delivers the field. As the label varies from run to run, the
	// test only checks that the error is sent once.
	// Issue: https://github.com/99designs/gqlgen/issues/4456
	t.Run("two labels", func(t *testing.T) {
		got := execute(t, es, `{ deferItem {
			id
			... @defer(label: "a") { slow }
			... @defer(label: "b") { failing }
		} }`)
		var responses []map[string]any
		require.NoError(t, json.Unmarshal([]byte(got), &responses))
		var errs []any
		for _, resp := range responses[1:] {
			if e, ok := resp["errors"].([]any); ok {
				errs = append(errs, e...)
				delete(resp, "errors")
			}
		}
		require.Equal(t, []any{map[string]any{
			"message": "failing field failed",
			"path":    []any{"deferItem", "failing"},
		}}, errs)
		rest, err := json.Marshal(responses)
		require.NoError(t, err)
		require.JSONEq(t, `[
			{"data":{"deferItem":{"id":1,"slow":null,"failing":null}},"hasNext":true},
			{"data":{"slow":"slow"},"label":"a","path":["deferItem"]},
			{"data":{"failing":null},"label":"b","path":["deferItem"]}
		]`, string(rest))
	})

	// KNOWN BUG: as in "deferred field", the first response holds the deferred fields
	// as null.
	// Expected: the first response leaves them out, for the same reason.
	// Issue: https://github.com/99designs/gqlgen/issues/4458
	t.Run("a null non-null field nulls its deferred group", func(t *testing.T) {
		got := execute(
			t,
			es,
			`{ deferItem { id ... @defer(label: "later") { slow nonNullFailing } } }`,
		)
		require.JSONEq(t, `[
			{"data":{"deferItem":{"id":1,"slow":null,"nonNullFailing":null}},"hasNext":true},
			{"data":null,"label":"later","path":["deferItem"],
				"errors":[{"message":"non-null field failed","path":["deferItem","nonNullFailing"]}]}
		]`, got)
	})

	t.Run("deferred fields of list elements", func(t *testing.T) {
		got := execute(t, es, `{ deferItems { id ... @defer(label: "later") { slow } } }`)
		require.JSONEq(t, `[
			{"data":{"deferItems":[{"id":1,"slow":null},{"id":2,"slow":null}]},"hasNext":true},
			{"data":{"slow":"slow"},"label":"later","path":["deferItems",0]},
			{"data":{"slow":"slow"},"label":"later","path":["deferItems",1]}
		]`, got)
	})
}

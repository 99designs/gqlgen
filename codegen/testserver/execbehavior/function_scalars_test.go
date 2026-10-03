package execbehavior

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFunctionScalars(t *testing.T) {
	resolvers := &Stub{}
	ptr := func(v string) *string {
		if v == "nil" {
			return nil
		}
		return &v
	}
	resolvers.QueryResolver.Maybe = func(ctx context.Context, v string) (*string, error) {
		return ptr(v), nil
	}
	resolvers.QueryResolver.MaybeNonNull = func(ctx context.Context, v string) (string, error) {
		return v, nil
	}
	resolvers.QueryResolver.CtxMaybe = func(ctx context.Context, v *string) (*string, error) {
		return v, nil
	}
	resolvers.QueryResolver.CtxMaybeNonNull = func(ctx context.Context, v string) (string, error) {
		return v, nil
	}
	srv := newServer(resolvers, DirectiveRoot{})

	// nullError is the error of a marshaler that returned null for the non-null field of
	// Query named field.
	nullError := func(field, typ string) string {
		return `"errors":[{"message":"cannot return null for non-null field Query.` + field +
			` (` + typ + `): the marshaler returned null for a non-nil value",`
	}
	for _, tc := range []struct{ name, query, want string }{
		{"value", `{ maybe(v: "x") }`, `{"data":{"maybe":"x"}}`},
		{"marshaler returns null", `{ maybe(v: "") }`, `{"data":{"maybe":null}}`},
		{"nil pointer", `{ maybe(v: "nil") }`, `{"data":{"maybe":null}}`},
		{
			"marshaler returns null for a non-null field",
			`{ maybeNonNull(v: "") }`,
			`{"data":null,` + nullError("maybeNonNull", "Maybe!") + `"path":["maybeNonNull"]}]}`,
		},
		{"context marshaler", `{ ctxMaybe(v: "y") }`, `{"data":{"ctxMaybe":"ctxMaybe:y"}}`},
		{"null argument", `{ ctxMaybe(v: null) }`, `{"data":{"ctxMaybe":null}}`},
		{"absent argument", `{ ctxMaybe }`, `{"data":{"ctxMaybe":null}}`},
		{"context marshaler returns null", `{ ctxMaybe(v: "") }`, `{"data":{"ctxMaybe":null}}`},
		{
			// KNOWN BUG: the error is reported, but the null does not propagate.
			// Expected: data is null, because the spec propagates a null in a non-null
			// field to the nearest nullable parent.
			// Issue: https://github.com/99designs/gqlgen/issues/4462
			"context marshaler returns null for a non-null field",
			`{ ctxMaybeNonNull(v: "") }`,
			`{"data":{"ctxMaybeNonNull":null},` + nullError("ctxMaybeNonNull", "CtxMaybe!") +
				`"path":["ctxMaybeNonNull"]}]}`,
		},
		{
			"context unmarshaler fails",
			`{ ctxMaybeNonNull(v: "bad") }`,
			`{"data":null,"errors":[{"message":"bad CtxMaybe","path":["ctxMaybeNonNull","v"]}]}`,
		},
		{
			"context unmarshaler fails for a pointer",
			`{ ctxMaybe(v: "bad") }`,
			`{"data":{"ctxMaybe":null},"errors":[{"message":"bad CtxMaybe","path":["ctxMaybe","v"]}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.JSONEq(t, tc.want, post(t, srv, tc.query))
		})
	}
}

package execbehavior

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

// TestUnmarshalNamedInput unmarshals the inputs of introspection.graphql,
// input_coercion.graphql and input_object_directive.graphql.
func TestUnmarshalNamedInput(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.UnmarshalNamed = func(ctx context.Context, kind string, raw map[string]any) (*string, error) {
		var s string
		switch kind {
		case "described":
			in, err := UnmarshalDescribedInput(ctx, raw)
			if err != nil {
				return nil, err
			}
			s = fmt.Sprintf("name=%v", *in.Name)
		case "coerce":
			in, err := UnmarshalCoerceInput(ctx, raw)
			if err != nil {
				return nil, err
			}
			s = fmt.Sprintf("numbers=%v items=%d", in.Numbers, len(in.Items))
		case "checked":
			in, err := UnmarshalCheckedInput(ctx, raw)
			if err != nil {
				return nil, err
			}
			s = in.Mode + ":" + in.Value
		case "unknown":
			var out any
			if err := graphql.UnmarshalNamedInputFromContext(
				ctx,
				"NoSuchInput",
				raw,
				&out,
			); err != nil {
				return nil, err
			}
		}
		return &s, nil
	}
	srv := newServer(resolvers, DirectiveRoot{
		InputCheck: func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
			res, err := next(ctx)
			if err != nil {
				return nil, err
			}
			in := res.(CheckedInput)
			in.Value = "checked " + in.Value
			return in, nil
		},
	})

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "defaults apply",
			query: `{ unmarshalNamed(kind: "described", raw: {}) }`,
			want:  `{"data":{"unmarshalNamed":"name=anonymous"}}`,
		},
		{
			name:  "values are coerced",
			query: `{ unmarshalNamed(kind: "coerce", raw: { numbers: 1, items: [{ value: "a" }] }) }`,
			want:  `{"data":{"unmarshalNamed":"numbers=[1] items=1"}}`,
		},
		{
			name:  "invalid nested value",
			query: `{ unmarshalNamed(kind: "coerce", raw: { items: [{ value: "bad" }] }) }`,
			want: `{"data":{"unmarshalNamed":null},"errors":[
				{"message":"strict rejects this value","path":["unmarshalNamed","items",0,"value"]}]}`,
		},
		{
			name:  "INPUT_OBJECT directives run",
			query: `{ unmarshalNamed(kind: "checked", raw: { mode: "m", value: "v" }) }`,
			want:  `{"data":{"unmarshalNamed":"m:checked v"}}`,
		},
		{
			name:  "unknown input",
			query: `{ unmarshalNamed(kind: "unknown", raw: {}) }`,
			want: `{"data":{"unmarshalNamed":null},"errors":[
				{"message":"graphql: no unmarshaler for input \"NoSuchInput\"; inputs bound to a Go type with its own UnmarshalGQL method are not indexed","path":["unmarshalNamed"]}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.JSONEq(t, tt.want, post(t, srv, tt.query))
		})
	}

	t.Run("outside a request", func(t *testing.T) {
		_, err := UnmarshalCoerceInput(context.Background(), map[string]any{})
		require.EqualError(t, err, "graphql: the input context is empty")
	})
}

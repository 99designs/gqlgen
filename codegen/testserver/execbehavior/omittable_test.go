package execbehavior

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

// describeOmittable tells whether an omittable value is absent, null or set.
func describeOmittable[T any](v graphql.Omittable[*T]) string {
	value, set := v.ValueOK()
	switch {
	case !set:
		return "absent"
	case value == nil:
		return "null"
	default:
		return fmt.Sprint(*value)
	}
}

func TestOmittableInputFields(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Omittable = func(ctx context.Context, input OmittableInput) (string, error) {
		list := "absent"
		if value, set := input.List.ValueOK(); set {
			list = "[" + strings.Join(value, ",") + "]"
			if value == nil {
				list = "null"
			}
		}
		text, count := describeOmittable(input.Text), describeOmittable(input.Count)
		return fmt.Sprintf("text=%s count=%s list=%s", text, count, list), nil
	}
	srv := newServer(resolvers, DirectiveRoot{})

	for _, tc := range []struct{ query, want string }{
		{`{ omittable(input: {}) }`, "text=absent count=absent list=absent"},
		{`{ omittable(input: {text: null, count: null, list: null}) }`, "text=null count=null list=null"},
		{`{ omittable(input: {text: "a", count: 2, list: ["x", "y"]}) }`, "text=a count=2 list=[x,y]"},
		{`{ omittable(input: {count: 0, list: []}) }`, "text=absent count=0 list=[]"},
	} {
		require.JSONEq(t, `{"data":{"omittable":"`+tc.want+`"}}`, post(t, srv, tc.query), tc.query)
	}

	// Variables that are absent or null behave like the literals.
	query := `query($text: String) { omittable(input: {text: $text}) }`
	require.JSONEq(
		t,
		`{"data":{"omittable":"text=absent count=absent list=absent"}}`,
		postVars(t, srv, query, nil),
	)
	require.JSONEq(
		t,
		`{"data":{"omittable":"text=null count=absent list=absent"}}`,
		postVars(t, srv, query, map[string]any{"text": nil}),
	)
}

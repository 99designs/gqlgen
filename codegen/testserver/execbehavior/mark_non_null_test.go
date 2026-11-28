package execbehavior

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

func TestMarkNonNull(t *testing.T) {
	// newMarkedServer returns a server that marks the field named marked non-null, whose
	// child holds value, and whose resolvedValue resolves to resolved and resolvedErr.
	newMarkedServer := func(
		marked string,
		value, resolved *string,
		resolvedErr error,
	) *handler.Server {
		resolvers := &Stub{}
		resolvers.QueryResolver.MarkedParent = func(ctx context.Context) (*MarkedParent, error) {
			name := "parent"
			return &MarkedParent{Name: &name, Child: &MarkedChild{Value: value}}, nil
		}
		resolvers.MarkedChildResolver.ResolvedValue = func(ctx context.Context, obj *MarkedChild) (*string, error) {
			return resolved, resolvedErr
		}
		resolvers.MarkedChildResolver.Node = func(ctx context.Context, obj *MarkedChild) (MarkedNode, error) {
			return nil, nil
		}
		srv := newServer(resolvers, DirectiveRoot{})
		srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (any, error) {
			if graphql.GetFieldContext(ctx).Field.Name == marked {
				graphql.MarkNonNull(ctx)
			}
			return next(ctx)
		})
		return srv
	}
	s := "s"
	// nilMarked is the error of a nil in the marked field of MarkedChild named field.
	nilMarked := func(field string) string {
		return `{"message":"cannot return null for non-null field MarkedChild.` + field +
			` (MarkedNode): the field was marked non-null at runtime and resolved to nil",` +
			`"path":["markedParent","child","` + field + `"]}`
	}

	tests := []struct {
		name        string
		marked      string
		value       *string
		resolved    *string
		resolvedErr error
		query       string
		want        string
	}{
		{
			name:  "unmarked null field stays null",
			value: nil,
			query: `{ markedParent { name child { value } } }`,
			want:  `{"data":{"markedParent":{"name":"parent","child":{"value":null}}}}`,
		},
		{
			name:   "marked field with a value",
			marked: "value",
			value:  &s,
			query:  `{ markedParent { name child { value } } }`,
			want:   `{"data":{"markedParent":{"name":"parent","child":{"value":"s"}}}}`,
		},
		{
			name:        "marked resolver field that fails nulls its parent",
			marked:      "resolvedValue",
			resolvedErr: errors.New("resolver failed"),
			query:       `{ markedParent { name child { resolvedValue } } }`,
			want: `{"data":{"markedParent":{"name":"parent","child":null}},"errors":[
				{"message":"resolver failed","path":["markedParent","child","resolvedValue"]}]}`,
		},
		{
			name:   "marked field resolving to a nil interface nulls its parent",
			marked: "node",
			query:  `{ markedParent { name child { node { id } } } }`,
			want: `{"data":{"markedParent":{"name":"parent","child":null}},"errors":[
				` + nilMarked("node") + `]}`,
		},
		{
			name:   "marked struct field holding a nil interface nulls its parent",
			marked: "plainNode",
			query:  `{ markedParent { name child { plainNode { id } } } }`,
			want: `{"data":{"markedParent":{"name":"parent","child":null}},"errors":[
				` + nilMarked("plainNode") + `]}`,
		},
		// KNOWN BUG: a marked field holding or returning a nil pointer is marshaled as
		// null without an error. The executor only sees a nil interface as nil, and the
		// pointer reaches it as a non-nil any.
		// Expected: an error, and the null propagates to the parent, because the
		// documentation of MarkNonNull says so for a marked field that resolves to nil.
		// Issue: https://github.com/99designs/gqlgen/issues/4457
		{
			name:   "marked struct field holding a nil pointer stays null",
			marked: "value",
			query:  `{ markedParent { name child { value } } }`,
			want:   `{"data":{"markedParent":{"name":"parent","child":{"value":null}}}}`,
		},
		{
			name:   "marked resolver field returning a nil pointer stays null",
			marked: "resolvedValue",
			query:  `{ markedParent { name child { resolvedValue } } }`,
			want:   `{"data":{"markedParent":{"name":"parent","child":{"resolvedValue":null}}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newMarkedServer(tt.marked, tt.value, tt.resolved, tt.resolvedErr)
			require.JSONEq(t, tt.want, post(t, srv, tt.query))
		})
	}
}

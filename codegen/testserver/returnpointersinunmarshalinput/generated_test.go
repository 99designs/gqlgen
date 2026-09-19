//go:generate rm -f resolver.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package returnpointersinunmarshalinput

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
)

// This package exists to exercise return_pointers_in_unmarshalinput, which
// inverts the generated unmarshalers to return a pointer. Before the nil-object
// early return in input.gotpl was fixed, the option produced a value where a
// pointer was declared and no schema using it compiled at all, so the option
// being generated and this package building is itself the regression test.
func TestUnmarshalInputReturnsPointer(t *testing.T) {
	name := "bob"
	limit := 7

	tests := []struct {
		name string
		obj  any
		want SearchFilters
	}{
		{
			name: "every field present",
			obj:  map[string]any{"name": name, "limit": limit},
			want: SearchFilters{Name: &name, Limit: &limit},
		},
		{
			name: "no fields present",
			obj:  map[string]any{},
			want: SearchFilters{},
		},
		{
			// A null input yields a pointer to the zero value, matching what the
			// value-returning configuration produces for the same input.
			name: "null input",
			obj:  nil,
			want: SearchFilters{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ec := &executionContext{}

			got, err := ec.unmarshalInputSearchFilters(context.Background(), tt.obj)

			require.NoError(t, err)
			require.NotNil(t, got, "the unmarshaler must never return a nil pointer")
			assert.Equal(t, tt.want, *got)
		})
	}
}

// With return_pointers_in_unmarshalinput the generated helper returns *T, and the caller
// sees that in its signature rather than discovering it through a failed lookup. Note
// that the option changes only the unmarshaler return: the resolver argument shape still
// follows schema nullability.
func TestGeneratedTypedUnmarshalerReturnsPointer(t *testing.T) {
	var (
		called bool
		got    *SearchFilters
		gotErr error
	)

	resolvers := &Stub{}
	resolvers.QueryResolver.Search = func(ctx context.Context, filters SearchFilters) (string, error) {
		called = true
		got, gotErr = UnmarshalSearchFilters(ctx, map[string]any{"name": "bob"})
		return "ok", nil
	}

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers}))

	var resp struct{ Search string }
	require.NoError(t, client.New(srv).Post(`query { search(filters: {name: "x"}) }`, &resp))

	require.True(t, called, "the resolver never ran, so nothing was exercised")
	require.NoError(t, gotErr)
	require.NotNil(t, got)
	require.NotNil(t, got.Name)
	assert.Equal(t, "bob", *got.Name)
}

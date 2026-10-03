//go:generate rm -f resolver.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package zerovaluenonnullobjectfields

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
)

// This package exercises zero_value_non_null_object_fields (see
// https://github.com/99designs/gqlgen/issues/3912): with the option on, a non-null
// object field resolved by direct struct field access falls back to a zero-value
// pointer instead of triggering GraphQL's null-propagation error when the underlying
// Go field is left nil. The option defaults to off, since per the GraphQL spec a null
// at a non-null position is an error regardless of how the value was produced; the
// codegen/testserver/singlefile and codegen/testserver/followschema nulls_test.go
// files cover that default (error) behavior.
func TestZeroValueNonNullObjectFields(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.MissingRequiredField = func(ctx context.Context) (*Parent, error) {
		return &Parent{ID: "p1"}, nil
	}

	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolvers}))
	c := client.New(srv)

	var resp struct {
		MissingRequiredField struct {
			ID    string
			Child struct{ Name string }
		}
	}
	err := c.Post(`query { missingRequiredField { id, child { name } } }`, &resp)

	require.NoError(t, err)
	require.Equal(t, "p1", resp.MissingRequiredField.ID)
	require.Empty(t, resp.MissingRequiredField.Child.Name)
}

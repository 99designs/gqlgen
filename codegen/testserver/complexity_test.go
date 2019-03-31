package testserver

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestComplexityCollisions(t *testing.T) {
	resolvers := &Stub{}

	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolvers})))
	c := client.New(srv.URL)

	resolvers.QueryResolver.Overlapping = func(ctx context.Context) (fields *OverlappingFields, e error) {
		return &OverlappingFields{
			Foo:    2,
			NewFoo: 3,
		}, nil
	}

	resolvers.OverlappingFieldsResolver.OldFoo = func(ctx context.Context, obj *OverlappingFields) (i int, e error) {
		return obj.Foo, nil
	}

	var resp struct {
		Overlapping struct {
			OneFoo  int `json:"oneFoo"`
			TwoFoo  int `json:"twoFoo"`
			OldFoo  int `json:"oldFoo"`
			NewFoo  int `json:"newFoo"`
			New_foo int `json:"new_foo"`
		}
	}
	c.MustPost(`query { overlapping { oneFoo, twoFoo, oldFoo, newFoo, new_foo } }`, &resp)
	require.Equal(t, 2, resp.Overlapping.OneFoo)
	require.Equal(t, 2, resp.Overlapping.TwoFoo)
	require.Equal(t, 2, resp.Overlapping.OldFoo)
	require.Equal(t, 3, resp.Overlapping.NewFoo)
	require.Equal(t, 3, resp.Overlapping.New_foo)

}

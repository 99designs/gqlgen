//go:generate rm -f resolver.go
//go:generate go run ../../testdata/gqlgen.go -stub stub.go

package testserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestGeneratedResolversAreValid(t *testing.T) {
	http.Handle("/query", handler.GraphQL(NewExecutableSchema(Config{
		Resolvers: &Resolver{},
	})))
}

func TestForcedResolverFieldIsPointer(t *testing.T) {
	field, ok := reflect.TypeOf((*ForcedResolverResolver)(nil)).Elem().MethodByName("Field")
	require.True(t, ok)
	require.Equal(t, "*testserver.Circle", field.Type.Out(0).String())
}

func TestEnums(t *testing.T) {
	t.Run("list of enums", func(t *testing.T) {
		require.Equal(t, StatusOk, AllStatus[0])
		require.Equal(t, StatusError, AllStatus[1])
	})
}

func TestUnionFragments(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.ShapeUnion = func(ctx context.Context) (ShapeUnion, error) {
		return &Circle{Radius: 32}, nil
	}

	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolvers})))
	c := client.New(srv.URL)

	t.Run("inline fragment on union", func(t *testing.T) {
		var resp struct {
			ShapeUnion struct {
				Radius float64
			}
		}
		c.MustPost(`query {
			shapeUnion {
				... on Circle {
					radius
				}
			}
		}
		`, &resp)
		require.NotEmpty(t, resp.ShapeUnion.Radius)
	})

	t.Run("named fragment", func(t *testing.T) {
		var resp struct {
			ShapeUnion struct {
				Radius float64
			}
		}
		c.MustPost(`query {
			shapeUnion {
				...C
			}
		}

		fragment C on ShapeUnion {
			... on Circle {
				radius
			}
		}
		`, &resp)
		require.NotEmpty(t, resp.ShapeUnion.Radius)
	})
}

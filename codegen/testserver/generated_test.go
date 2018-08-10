//go:generate rm -f resolver.go
//go:generate gorunpkg github.com/99designs/gqlgen

package testserver

import (
	"net/http"
	"testing"

	"reflect"

	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestCompiles(t *testing.T) {
	http.Handle("/query", handler.GraphQL(NewExecutableSchema(Config{
		Resolvers: &Resolver{},
	})))
}

func TestForcedResolverFieldIsPointer(t *testing.T) {
	field, ok := reflect.TypeOf((*ForcedResolverResolver)(nil)).Elem().MethodByName("Field")
	require.True(t, ok)
	require.Equal(t, "*testserver.Circle", field.Type.Out(0).String())
}

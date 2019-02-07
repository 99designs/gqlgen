//go:generate rm -f resolver.go
//go:generate go run ../../testdata/gqlgen.go -stub stub.go

package testserver

import (
	"net/http"
	"reflect"
	"testing"

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

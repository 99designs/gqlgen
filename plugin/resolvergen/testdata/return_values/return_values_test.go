package return_values

import (
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

//go:generate rm -f resolvers.go
//go:generate go run ../../../../testdata/gqlgen.go -config gqlgen.yml

func TestResolverReturnTypes(t *testing.T) {
	// verify that the return value of the User resolver is a struct, not a pointer
	require.Equal(t, "struct", reflect.TypeOf((&queryResolver{}).User).Out(0).Kind().String())
	// the UserPointer resolver should return a pointer
	require.Equal(t, "ptr", reflect.TypeOf((&queryResolver{}).UserPointer).Out(0).Kind().String())
}

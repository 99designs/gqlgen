package testserver

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInterfaces(t *testing.T) {
	t.Run("slices of interfaces are not pointers", func(t *testing.T) {
		field, ok := reflect.TypeOf((*QueryResolver)(nil)).Elem().MethodByName("Shapes")
		require.True(t, ok)
		require.Equal(t, "[]testserver.Shape", field.Type.Out(0).String())
	})
}

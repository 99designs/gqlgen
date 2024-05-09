package followschema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserPtr(t *testing.T) {
	s := &Stub{}
	r := reflect.TypeOf(s.QueryResolver.OptionalUnion)
	require.Equal(t, reflect.Interface, r.Out(0).Kind())
}

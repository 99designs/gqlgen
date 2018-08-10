package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeterministicDecollisioning(t *testing.T) {
	a := Imports{
		imports: []*Import{
			{Name: "types", Path: "foobar/types"},
			{Name: "types", Path: "bazfoo/types"},
		},
	}.finalize()

	b := Imports{
		imports: []*Import{
			{Name: "types", Path: "bazfoo/types"},
			{Name: "types", Path: "foobar/types"},
		},
	}.finalize()

	require.EqualValues(t, a, b)
}

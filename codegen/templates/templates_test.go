package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToUpper(t *testing.T) {
	require.Equal(t, "ToCamel", ToCamel("TO_CAMEL"))
	require.Equal(t, "ToCamel", ToCamel("to_camel"))
	require.Equal(t, "ToCamel", ToCamel("toCamel"))
	require.Equal(t, "ToCamel", ToCamel("ToCamel"))
	require.Equal(t, "ToCamel", ToCamel("to-camel"))
}

func TestCenter(t *testing.T) {
	require.Equal(t, "fffff", center(3, "#", "fffff"))
	require.Equal(t, "##fffff###", center(10, "#", "fffff"))
	require.Equal(t, "###fffff###", center(11, "#", "fffff"))
}

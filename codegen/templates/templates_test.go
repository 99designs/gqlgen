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

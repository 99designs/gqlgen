package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToCamel(t *testing.T) {
	require.Equal(t, "ToCamel", ToCamel("TO_CAMEL"))
	require.Equal(t, "ToCamel", ToCamel("to_camel"))
	require.Equal(t, "ToCamel", ToCamel("toCamel"))
	require.Equal(t, "ToCamel", ToCamel("ToCamel"))
	require.Equal(t, "ToCamel", ToCamel("to-camel"))

	require.Equal(t, "RelatedURLs", ToCamel("RelatedURLs"))
	require.Equal(t, "ImageIDs", ToCamel("ImageIDs"))
	require.Equal(t, "FooID", ToCamel("FooID"))
	require.Equal(t, "IDFoo", ToCamel("IDFoo"))
	require.Equal(t, "FooASCII", ToCamel("FooASCII"))
	require.Equal(t, "ASCIIFoo", ToCamel("ASCIIFoo"))
	require.Equal(t, "FooUTF8", ToCamel("FooUTF8"))
	require.Equal(t, "UTF8Foo", ToCamel("UTF8Foo"))

	require.Equal(t, "A", ToCamel("A"))
	require.Equal(t, "ID", ToCamel("ID"))
	require.Equal(t, "", ToCamel(""))
}

func TestCenter(t *testing.T) {
	require.Equal(t, "fffff", center(3, "#", "fffff"))
	require.Equal(t, "##fffff###", center(10, "#", "fffff"))
	require.Equal(t, "###fffff###", center(11, "#", "fffff"))
}

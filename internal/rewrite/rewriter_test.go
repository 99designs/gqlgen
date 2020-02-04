package rewrite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriter(t *testing.T) {
	r, err := New("github.com/99designs/gqlgen/internal/rewrite/testdata")
	require.NoError(t, err)

	body := r.GetMethodBody("Foo", "Method")
	require.Equal(t, `
	// leading comment

	// field comment
	m.Field++

	// trailing comment
`, body)

	imps := r.ExistingImports("testdata/example.go")
	require.Len(t, imps, 2)
	assert.Equal(t, []Import{
		{
			Alias:      "",
			ImportPath: "fmt",
		},
		{
			Alias:      "lol",
			ImportPath: "bytes",
		},
	}, imps)
}

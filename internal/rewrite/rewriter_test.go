package rewrite

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriter(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		r, err := New("testdata")
		require.NoError(t, err)

		body := r.GetMethodBody("Foo", "Method")
		require.Equal(t, `
	// leading comment

	// field comment
	m.Field++

	// trailing comment
`, strings.ReplaceAll(body, "\r\n", "\n"))

		imps := r.ExistingImports("testdata/example.go")
		require.Len(t, imps, 2)
		assert.Equal(t, []Import{
			{
				Alias:      "lol",
				ImportPath: "bytes",
			},
			{
				Alias:      "",
				ImportPath: "fmt",
			},
		}, imps)
	})

	t.Run("out of scope dir returns no-op rewriter", func(t *testing.T) {
		r, err := New("../../../out-of-gomod/package")
		require.NoError(t, err)
		require.NotNil(t, r)
		// No-op rewriter should return empty results without panicking
		assert.Nil(t, r.GetPrevDecl("Foo", "Bar"))
		assert.Equal(t, "", r.GetMethodBody("Foo", "Bar"))
		assert.Equal(t, "", r.GetMethodComment("Foo", "Bar"))
		assert.Equal(t, "", r.RemainingSource("nonexistent.go"))
		assert.Nil(t, r.ExistingImports("nonexistent.go"))
	})

	t.Run("nonexistent dir returns no-op rewriter", func(t *testing.T) {
		r, err := New("/tmp/definitely-does-not-exist-gqlgen-test")
		require.NoError(t, err)
		require.NotNil(t, r)
		assert.Nil(t, r.GetPrevDecl("X", "Y"))
	})
}

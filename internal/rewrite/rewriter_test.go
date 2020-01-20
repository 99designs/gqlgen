package rewrite

import (
	"testing"

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
}

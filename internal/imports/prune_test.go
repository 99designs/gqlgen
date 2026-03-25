package imports

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/internal/code"
)

func TestPrune(t *testing.T) {
	testFile := "testdata/unused.go"
	expectedFile := "testdata/unused.expected.go"

	t.Run("default behavior (imports.Process)", func(t *testing.T) {
		b, err := Prune(testFile, mustReadFile(testFile), code.NewPackages(), PruneOptions{})
		require.NoError(t, err)
		require.Equal(
			t,
			strings.ReplaceAll(string(mustReadFile(expectedFile)), "\r\n", "\n"),
			string(b),
		)
	})

	t.Run("with skip_import_grouping", func(t *testing.T) {
		b, err := Prune(testFile, mustReadFile(testFile), code.NewPackages(), PruneOptions{
			SkipImportGrouping: true,
		})
		require.NoError(t, err)
		require.Contains(t, string(b), "package testdata")
	})

	t.Run("with buffer_pooling only", func(t *testing.T) {
		b, err := Prune(testFile, mustReadFile(testFile), code.NewPackages(), PruneOptions{
			UseBufferPooling: true,
		})
		require.NoError(t, err)
		require.Equal(
			t,
			strings.ReplaceAll(string(mustReadFile(expectedFile)), "\r\n", "\n"),
			string(b),
		)
	})

	t.Run("with both options", func(t *testing.T) {
		b, err := Prune(testFile, mustReadFile(testFile), code.NewPackages(), PruneOptions{
			SkipImportGrouping: true,
			UseBufferPooling:   true,
		})
		require.NoError(t, err)
		require.Contains(t, string(b), "package testdata")
	})
}

func mustReadFile(filename string) []byte {
	b, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return b
}

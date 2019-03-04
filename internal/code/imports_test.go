package code

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportPathForDir(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	assert.Equal(t, "github.com/99designs/gqlgen/internal/code", ImportPathForDir(wd))
	assert.Equal(t, "github.com/99designs/gqlgen/api", ImportPathForDir(filepath.Join(wd, "..", "..", "api")))

	// doesnt contain go code, but should still give a valid import path
	assert.Equal(t, "github.com/99designs/gqlgen/docs", ImportPathForDir(filepath.Join(wd, "..", "..", "docs")))

	// directory does not exist
	assert.Equal(t, "github.com/99designs/gqlgen/dos", ImportPathForDir(filepath.Join(wd, "..", "..", "dos")))
}

func TestNameForPackage(t *testing.T) {
	assert.Equal(t, "api", NameForPackage("github.com/99designs/gqlgen/api"))

	// does not contain go code, should still give a valid name
	assert.Equal(t, "docs", NameForPackage("github.com/99designs/gqlgen/docs"))
	assert.Equal(t, "github_com", NameForPackage("github.com"))
}

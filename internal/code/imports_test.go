package code

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
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

	if runtime.GOOS == "windows" {
		assert.Equal(t, "", ImportPathForDir("C:/doesnotexist"))
	} else {
		assert.Equal(t, "", ImportPathForDir("/doesnotexist"))
	}
}

func TestNameForPackage(t *testing.T) {
	testPkg1 := "github.com/99designs/gqlgen/api"
	testPkg2 := "github.com/99designs/gqlgen/docs"
	testPkg3 := "github.com"
	ps, err := packages.Load(nil, testPkg1, testPkg2, testPkg3)
	require.NoError(t, err)
	RecordPackagesList(ps)
	assert.Equal(t, "api", NameForPackage(testPkg1))

	// does not contain go code, should still give a valid name
	assert.Equal(t, "docs", NameForPackage(testPkg2))
	assert.Equal(t, "github_com", NameForPackage(testPkg3))
}

func TestNameForDir(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	assert.Equal(t, "tmp", NameForDir("/tmp"))
	assert.Equal(t, "code", NameForDir(wd))
	assert.Equal(t, "internal", NameForDir(wd+"/.."))
	assert.Equal(t, "main", NameForDir(wd+"/../.."))
}

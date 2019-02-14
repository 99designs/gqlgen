package gopath

import (
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	origBuildContext := build.Default
	defer func() { build.Default = origBuildContext }()

	// Make a temporary directory and add a go.mod file for the package 'foo'
	fooDir, err := ioutil.TempDir("", "gopath")
	assert.Nil(t, err)
	defer func() { err := os.RemoveAll(fooDir); assert.Nil(t, err) }()
	err = ioutil.WriteFile(filepath.Join(fooDir, "go.mod"), []byte("module foo\n\nrequire ()"), 0644)
	assert.Nil(t, err)

	if runtime.GOOS == "windows" {
		build.Default.GOPATH = `C:\go;C:\Users\user\go`

		assert.True(t, Contains(`C:\go\src\github.com\vektah\gqlgen`))
		assert.True(t, Contains(`C:\go\src\fpp`))
		assert.True(t, Contains(`C:/go/src/github.com/vektah/gqlgen`))
		assert.True(t, Contains(`C:\Users\user\go\src\foo`))
		assert.False(t, Contains(`C:\tmp`))
		assert.False(t, Contains(`C:\Users\user`))
		assert.False(t, Contains(`C:\Users\another\go`))

		// C:/Users/someone/AppData/Local/Temp/gopath123456/bar
		assert.True(t, Contains(filepath.Join(fooDir, "bar")))

	} else {
		build.Default.GOPATH = "/go:/home/user/go"

		assert.True(t, Contains("/go/src/github.com/vektah/gqlgen"))
		assert.True(t, Contains("/go/src/foo"))
		assert.True(t, Contains("/home/user/go/src/foo"))
		assert.False(t, Contains("/tmp"))
		assert.False(t, Contains("/home/user"))
		assert.False(t, Contains("/home/another/go"))

		// /tmp/gopath123456/bar
		assert.True(t, Contains(filepath.Join(fooDir, "bar")))
	}
}

func TestDir2Package(t *testing.T) {
	origBuildContext := build.Default
	defer func() { build.Default = origBuildContext }()

	// Make a temporary directory and add a go.mod file for the package 'foo'
	fooDir, err := ioutil.TempDir("", "gopath")
	assert.Nil(t, err)
	defer func() { err := os.RemoveAll(fooDir); assert.Nil(t, err) }()
	err = ioutil.WriteFile(filepath.Join(fooDir, "go.mod"), []byte("module foo\n\nrequire ()"), 0644)
	assert.Nil(t, err)

	if runtime.GOOS == "windows" {
		build.Default.GOPATH = "C:/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx;C:/a/y;C:/b/"

		assert.Equal(t, "foo/bar", MustDir2Import("C:/a/y/src/foo/bar"))
		assert.Equal(t, "foo/bar", MustDir2Import(`C:\a\y\src\foo\bar`))
		assert.Equal(t, "foo/bar", MustDir2Import("C:/b/src/foo/bar"))
		assert.Equal(t, "foo/bar", MustDir2Import(`C:\b\src\foo\bar`))

		assert.PanicsWithValue(t, NotFound, func() {
			MustDir2Import("C:/tmp/foo")
		})

		// C:/Users/someone/AppData/Local/Temp/gopath123456/bar
		assert.Equal(t, "foo/bar", MustDir2Import(filepath.Join(fooDir, "bar")))
	} else {
		build.Default.GOPATH = "/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx:/a/y:/b/"

		assert.Equal(t, "foo/bar", MustDir2Import("/a/y/src/foo/bar"))
		assert.Equal(t, "foo/bar", MustDir2Import("/b/src/foo/bar"))

		assert.PanicsWithValue(t, NotFound, func() {
			MustDir2Import("/tmp/foo")
		})

		// /tmp/gopath123456/bar
		assert.Equal(t, "foo/bar", MustDir2Import(filepath.Join(fooDir, "bar")))
	}
}

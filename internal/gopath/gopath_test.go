package gopath

import (
	"go/build"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains(t *testing.T) {
	origBuildContext := build.Default
	defer func() { build.Default = origBuildContext }()

	if runtime.GOOS == "windows" {
		build.Default.GOPATH = `C:\go;C:\Users\user\go`

		assert.True(t, Contains(`C:\go\src\github.com\vektah\gqlgen`))
		assert.True(t, Contains(`C:\go\src\fpp`))
		assert.True(t, Contains(`C:/go/src/github.com/vektah/gqlgen`))
		assert.True(t, Contains(`C:\Users\user\go\src\foo`))
		assert.False(t, Contains(`C:\tmp`))
		assert.False(t, Contains(`C:\Users\user`))
		assert.False(t, Contains(`C:\Users\another\go`))
	} else {
		build.Default.GOPATH = "/go:/home/user/go"

		assert.True(t, Contains("/go/src/github.com/vektah/gqlgen"))
		assert.True(t, Contains("/go/src/foo"))
		assert.True(t, Contains("/home/user/go/src/foo"))
		assert.False(t, Contains("/tmp"))
		assert.False(t, Contains("/home/user"))
		assert.False(t, Contains("/home/another/go"))
	}
}

func TestDir2Package(t *testing.T) {
	origBuildContext := build.Default
	defer func() { build.Default = origBuildContext }()

	if runtime.GOOS == "windows" {
		build.Default.GOPATH = "C:/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx;C:/a/y;C:/b/"

		assert.Equal(t, "foo/bar", MustDir2Import("C:/a/y/src/foo/bar"))
		assert.Equal(t, "foo/bar", MustDir2Import(`C:\a\y\src\foo\bar`))
		assert.Equal(t, "foo/bar", MustDir2Import("C:/b/src/foo/bar"))
		assert.Equal(t, "foo/bar", MustDir2Import(`C:\b\src\foo\bar`))

		assert.PanicsWithValue(t, NotFound, func() {
			MustDir2Import("C:/tmp/foo")
		})
	} else {
		build.Default.GOPATH = "/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx:/a/y:/b/"

		assert.Equal(t, "foo/bar", MustDir2Import("/a/y/src/foo/bar"))
		assert.Equal(t, "foo/bar", MustDir2Import("/b/src/foo/bar"))

		assert.PanicsWithValue(t, NotFound, func() {
			MustDir2Import("/tmp/foo")
		})
	}
}

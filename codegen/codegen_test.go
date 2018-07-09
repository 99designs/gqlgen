package codegen

import (
	"go/build"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_fullPackageName(t *testing.T) {
	origBuildContext := build.Default
	defer func() { build.Default = origBuildContext }()

	t.Run("gopath longer than package name", func(t *testing.T) {
		build.Default.GOPATH = "/a/src/xxxxxxxxxxxxxxxxxxxxxxxx:/b/src/y"
		var got string
		ok := assert.NotPanics(t, func() { got = importPath("/b/src/y/foo/bar") })
		if ok {
			assert.Equal(t, "/b/src/y/foo/bar", got)
		}
	})
	t.Run("stop searching on first hit", func(t *testing.T) {
		build.Default.GOPATH = "/a/src/x:/b/src/y"

		var got string
		ok := assert.NotPanics(t, func() { got = importPath("/a/src/x/foo/bar") })
		if ok {
			assert.Equal(t, "/a/src/x/foo/bar", got)
		}
	})
}

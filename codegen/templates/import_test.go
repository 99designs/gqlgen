package templates

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImports(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	aBar := "github.com/99designs/gqlgen/codegen/templates/testdata/a/bar"
	bBar := "github.com/99designs/gqlgen/codegen/templates/testdata/b/bar"
	mismatch := "github.com/99designs/gqlgen/codegen/templates/testdata/pkg_mismatch"

	t.Run("multiple lookups is ok", func(t *testing.T) {
		a := Imports{destDir: wd}

		require.Equal(t, "bar", a.Lookup(aBar))
		require.Equal(t, "bar", a.Lookup(aBar))
	})

	t.Run("duplicates are decollisioned", func(t *testing.T) {
		a := Imports{destDir: wd}

		require.Equal(t, "bar", a.Lookup(aBar))
		require.Equal(t, "bar1", a.Lookup(bBar))

		t.Run("additionial calls get decollisioned name", func(t *testing.T) {
			require.Equal(t, "bar1", a.Lookup(bBar))
		})
	})

	t.Run("package name defined in code will be used", func(t *testing.T) {
		a := Imports{destDir: wd}

		require.Equal(t, "turtles", a.Lookup(mismatch))
	})

	t.Run("string printing for import block", func(t *testing.T) {
		a := Imports{destDir: wd}
		a.Lookup(aBar)
		a.Lookup(bBar)
		a.Lookup(mismatch)

		require.Equal(
			t,
			`"github.com/99designs/gqlgen/codegen/templates/testdata/a/bar"
bar1 "github.com/99designs/gqlgen/codegen/templates/testdata/b/bar"
"github.com/99designs/gqlgen/codegen/templates/testdata/pkg_mismatch"`,
			a.String(),
		)
	})

	t.Run("reserved collisions on path will panic", func(t *testing.T) {
		a := Imports{destDir: wd}

		a.Reserve(aBar)

		require.Panics(t, func() {
			a.Reserve(aBar)
		})
	})

	t.Run("reserved collisions on alias will panic", func(t *testing.T) {
		a := Imports{destDir: wd}

		a.Reserve(aBar)

		require.Panics(t, func() {
			a.Reserve(bBar)
		})
	})

	t.Run("aliased imports will not collide", func(t *testing.T) {
		a := Imports{destDir: wd}

		a.Reserve(aBar, "abar")
		a.Reserve(bBar, "bbar")

		require.Equal(t, `abar "github.com/99designs/gqlgen/codegen/templates/testdata/a/bar"
bbar "github.com/99designs/gqlgen/codegen/templates/testdata/b/bar"`, a.String())
	})
}

package templates

import (
	"fmt"
	"go/types"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/internal/code"
)

func TestImports(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	aBar := "github.com/99designs/gqlgen/codegen/templates/testdata/a/bar"
	bBar := "github.com/99designs/gqlgen/codegen/templates/testdata/b/bar"
	mismatch := "github.com/99designs/gqlgen/codegen/templates/testdata/pkg_mismatch"

	t.Run("multiple lookups is ok", func(t *testing.T) {
		a := Imports{destDir: wd, packages: code.NewPackages()}

		require.Equal(t, "bar", a.Lookup(aBar))
		require.Equal(t, "bar", a.Lookup(aBar))
	})

	t.Run("lookup by type", func(t *testing.T) {
		a := Imports{destDir: wd, packages: code.NewPackages()}

		pkg := types.NewPackage("github.com/99designs/gqlgen/codegen/templates/testdata/b/bar", "bar")
		typ := types.NewNamed(types.NewTypeName(0, pkg, "Boolean", types.Typ[types.Bool]), types.Typ[types.Bool], nil)

		require.Equal(t, "bar.Boolean", a.LookupType(typ))
	})

	t.Run("duplicates are decollisioned", func(t *testing.T) {
		a := Imports{destDir: wd, packages: code.NewPackages()}

		require.Equal(t, "bar", a.Lookup(aBar))
		require.Equal(t, "bar1", a.Lookup(bBar))

		t.Run("additionial calls get decollisioned name", func(t *testing.T) {
			require.Equal(t, "bar1", a.Lookup(bBar))
		})
	})

	t.Run("duplicates above 10 are decollisioned", func(t *testing.T) {
		a := Imports{destDir: wd, packages: code.NewPackages()}
		for i := 0; i < 100; i++ {
			cBar := fmt.Sprintf("github.com/99designs/gqlgen/codegen/templates/testdata/%d/bar", i)
			if i > 0 {
				require.Equal(t, fmt.Sprintf("bar%d", i), a.Lookup(cBar))
			} else {
				require.Equal(t, "bar", a.Lookup(cBar))
			}
		}
	})

	t.Run("package name defined in code will be used", func(t *testing.T) {
		a := Imports{destDir: wd, packages: code.NewPackages()}

		require.Equal(t, "turtles", a.Lookup(mismatch))
	})

	t.Run("string printing for import block", func(t *testing.T) {
		a := Imports{destDir: wd, packages: code.NewPackages()}
		a.Lookup(aBar)
		a.Lookup(bBar)
		a.Lookup(mismatch)

		require.Equal(
			t,
			`"github.com/99designs/gqlgen/codegen/templates/testdata/a/bar"
bar1 "github.com/99designs/gqlgen/codegen/templates/testdata/b/bar"
turtles "github.com/99designs/gqlgen/codegen/templates/testdata/pkg_mismatch"`,
			a.String(),
		)
	})

	t.Run("aliased imports will not collide", func(t *testing.T) {
		a := Imports{destDir: wd, packages: code.NewPackages()}

		_, _ = a.Reserve(aBar, "abar")
		_, _ = a.Reserve(bBar, "bbar")

		require.Equal(t, `abar "github.com/99designs/gqlgen/codegen/templates/testdata/a/bar"
bbar "github.com/99designs/gqlgen/codegen/templates/testdata/b/bar"`, a.String())
	})
}

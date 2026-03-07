package code

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestPackages(t *testing.T) {
	t.Run("name for existing package does not load again", func(t *testing.T) {
		p := initialState(t)
		require.Equal(t, "a", p.NameForPackage("github.com/99designs/gqlgen/internal/code/testdata/a"))
		require.Equal(t, 1, p.numLoadCalls)
	})

	t.Run("name for unknown package makes name only load", func(t *testing.T) {
		p := initialState(t)
		require.Equal(t, "c", p.NameForPackage("github.com/99designs/gqlgen/internal/code/testdata/c"))
		require.Equal(t, 1, p.numLoadCalls)
		require.Equal(t, 1, p.numNameCalls)
	})

	t.Run("evicting a package causes it to load again", func(t *testing.T) {
		p := initialState(t)
		p.Evict("github.com/99designs/gqlgen/internal/code/testdata/b")
		require.Equal(t, "a", p.Load("github.com/99designs/gqlgen/internal/code/testdata/a").Name)
		require.Equal(t, 1, p.numLoadCalls)
		require.Equal(t, "b", p.Load("github.com/99designs/gqlgen/internal/code/testdata/b").Name)
		require.Equal(t, 2, p.numLoadCalls)
	})

	t.Run("able to load private package with build tags", func(t *testing.T) {
		p := initialState(t, WithBuildTags("private"))
		p.Evict("github.com/99designs/gqlgen/internal/code/testdata/a")
		require.Equal(t, "a", p.Load("github.com/99designs/gqlgen/internal/code/testdata/a").Name)
		require.Equal(t, 2, p.numLoadCalls)
		require.Equal(t, "p", p.Load("github.com/99designs/gqlgen/internal/code/testdata/p").Name)
		require.Equal(t, 3, p.numLoadCalls)
	})
}

func TestPackagesErrors(t *testing.T) {
	loadFirstErr := errors.New("first")
	loadSecondErr := errors.New("second")
	packageErr := packages.Error{Msg: "package"}
	p := &Packages{
		loadErrors: []error{loadFirstErr, loadSecondErr},
		packages: map[string]*packages.Package{"github.com/99designs/gqlgen/internal/code/testdata/a": {
			Errors: []packages.Error{packageErr},
		}},
	}

	errs := p.Errors()

	assert.Equal(t, PkgErrors([]error{loadFirstErr, loadSecondErr, packageErr}), errs)
}

func TestNameForPackage(t *testing.T) {
	var p Packages

	assert.Equal(t, "api", p.NameForPackage("github.com/99designs/gqlgen/api"))

	// does not contain go code, should still give a valid name
	assert.Equal(t, "docs", p.NameForPackage("github.com/99designs/gqlgen/docs"))
	assert.Equal(t, "github_com", p.NameForPackage("github.com"))
}

func TestLoadAllNames(t *testing.T) {
	var p Packages

	p.LoadAllNames("github.com/99designs/gqlgen/api", "github.com/99designs/gqlgen/docs", "github.com")

	// should now be cached
	assert.Equal(t, 0, p.numNameCalls)
	assert.Equal(t, "api", p.importToName["github.com/99designs/gqlgen/api"])
	assert.Equal(t, "docs", p.importToName["github.com/99designs/gqlgen/docs"])
	assert.Equal(t, "github_com", p.importToName["github.com"])
}

func TestInject(t *testing.T) {
	t.Run("inject sets sandbox mode and caches package", func(t *testing.T) {
		p := NewPackages()
		require.False(t, p.HasInjected())

		pkg := &packages.Package{
			ID:      "example.com/foo",
			Name:    "foo",
			PkgPath: "example.com/foo",
		}
		p.Inject("example.com/foo", pkg)

		require.True(t, p.HasInjected())
		require.True(t, p.noExternalLoad)
		loaded := p.Load("example.com/foo")
		require.NotNil(t, loaded)
		assert.Equal(t, "foo", loaded.Name)
	})

	t.Run("inject populates name cache", func(t *testing.T) {
		p := NewPackages()
		pkg := &packages.Package{
			ID:      "example.com/bar",
			Name:    "bar",
			PkgPath: "example.com/bar",
		}
		p.Inject("example.com/bar", pkg)
		assert.Equal(t, "bar", p.NameForPackage("example.com/bar"))
	})

	t.Run("sandbox mode skips packages.Load", func(t *testing.T) {
		p := NewPackages()
		pkg := &packages.Package{
			ID:      "example.com/foo",
			Name:    "foo",
			PkgPath: "example.com/foo",
		}
		p.Inject("example.com/foo", pkg)

		// LoadAll for a missing package should not call packages.Load
		pkgs := p.LoadAll("example.com/foo", "example.com/missing")
		assert.Equal(t, 0, p.numLoadCalls)
		assert.Equal(t, "foo", pkgs[0].Name)
		assert.Nil(t, pkgs[1])
	})

	t.Run("sandbox mode derives name from import path", func(t *testing.T) {
		p := NewPackages()
		pkg := &packages.Package{
			ID:      "example.com/foo",
			Name:    "foo",
			PkgPath: "example.com/foo",
		}
		p.Inject("example.com/foo", pkg)

		// Name for an unknown package in sandbox mode should be derived from path
		assert.Equal(t, "somepkg", p.NameForPackage("example.com/somepkg"))
		assert.Equal(t, 0, p.numLoadCalls)
	})
}

func TestCleanupUserPackagesWithInjected(t *testing.T) {
	t.Run("injected packages survive cleanup without prefix", func(t *testing.T) {
		p := NewPackages()
		injected := &packages.Package{ID: "example.com/injected", Name: "injected", PkgPath: "example.com/injected"}
		p.Inject("example.com/injected", injected)

		// Manually add a non-injected package
		p.packages["example.com/regular"] = &packages.Package{ID: "example.com/regular", Name: "regular", PkgPath: "example.com/regular"}

		p.CleanupUserPackages()

		assert.NotNil(t, p.packages["example.com/injected"])
		assert.Nil(t, p.packages["example.com/regular"])
	})

	t.Run("injected packages survive cleanup with prefix", func(t *testing.T) {
		p := NewPackages(PackagePrefixToCache("github.com/99designs/gqlgen/graphql"))
		injected := &packages.Package{ID: "example.com/injected", Name: "injected", PkgPath: "example.com/injected"}
		p.Inject("example.com/injected", injected)

		// Add a package matching the prefix (should survive) and one that doesn't (should be removed)
		p.packages["github.com/99designs/gqlgen/graphql/foo"] = &packages.Package{
			ID: "github.com/99designs/gqlgen/graphql/foo", Name: "foo",
			PkgPath: "github.com/99designs/gqlgen/graphql/foo",
		}
		p.packages["example.com/other"] = &packages.Package{ID: "example.com/other", Name: "other", PkgPath: "example.com/other"}

		p.CleanupUserPackages()

		assert.NotNil(t, p.packages["example.com/injected"], "injected should survive")
		assert.NotNil(t, p.packages["github.com/99designs/gqlgen/graphql/foo"], "prefix match should survive")
		assert.Nil(t, p.packages["example.com/other"], "non-matching, non-injected should be removed")
	})
}

func TestLoadWithTypesInSandboxMode(t *testing.T) {
	p := NewPackages()
	pkg := &packages.Package{
		ID:      "example.com/typed",
		Name:    "typed",
		PkgPath: "example.com/typed",
	}
	p.Inject("example.com/typed", pkg)

	// LoadWithTypes should return the cached package without calling packages.Load
	result := p.LoadWithTypes("example.com/typed")
	assert.Equal(t, "typed", result.Name)
	assert.Equal(t, 0, p.numLoadCalls)

	// For a missing package, should not panic or call packages.Load
	result = p.LoadWithTypes("example.com/missing")
	assert.Nil(t, result)
	assert.Equal(t, 0, p.numLoadCalls)
}

func initialState(t *testing.T, opts ...Option) *Packages {
	p := NewPackages(opts...)
	pkgs := p.LoadAll(
		"github.com/99designs/gqlgen/internal/code/testdata/a",
		"github.com/99designs/gqlgen/internal/code/testdata/b",
	)

	require.Empty(t, p.Errors())
	require.Equal(t, 1, p.numLoadCalls)
	require.Equal(t, 0, p.numNameCalls)
	require.Equal(t, "a", pkgs[0].Name)
	require.Equal(t, "b", pkgs[1].Name)
	return p
}

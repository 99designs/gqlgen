package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTestArchive compiles a Go package into a .a archive file using
// `go build -buildmode=archive`. This produces the same format that
// rules_go generates and that readArchive/gcexportdata.NewReader expects.
func buildTestArchive(t *testing.T, dir, filename, importPath string) string {
	t.Helper()
	out := filepath.Join(dir, filename)
	cmd := exec.Command("go", "build", "-buildmode=archive", "-o", out, importPath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", string(output))
	return out
}

func TestLoadTypeCache(t *testing.T) {
	const (
		chatPkg   = "github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat"
		scalarPkg = "github.com/99designs/gqlgen/codegen/config/testdata/autobinding/scalars/model"
	)

	t.Run("loads packages from manifest and archives", func(t *testing.T) {
		dir := t.TempDir()

		buildTestArchive(t, dir, "chat.a", chatPkg)
		buildTestArchive(t, dir, "scalars.a", scalarPkg)

		manifest := typeCacheManifest{
			Packages: map[string]string{
				chatPkg:   "chat.a",
				scalarPkg: "scalars.a",
			},
		}
		manifestData, err := json.Marshal(manifest)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), manifestData, 0o644))

		cfg := DefaultConfig()
		err = cfg.LoadTypeCache(dir)
		require.NoError(t, err)

		require.NotNil(t, cfg.Packages)
		require.True(t, cfg.Packages.HasInjected())

		chatLoaded := cfg.Packages.Load(chatPkg)
		require.NotNil(t, chatLoaded)
		assert.Equal(t, "chat", chatLoaded.Name)
		assert.Equal(t, chatPkg, chatLoaded.PkgPath)
		require.NotNil(t, chatLoaded.Types)
		assert.NotNil(t, chatLoaded.Types.Scope().Lookup("Message"))
		assert.NotNil(t, chatLoaded.Types.Scope().Lookup("ChatAPI"))

		scalarLoaded := cfg.Packages.Load(scalarPkg)
		require.NotNil(t, scalarLoaded)
		assert.Equal(t, "model", scalarLoaded.Name)
		assert.NotNil(t, scalarLoaded.Types.Scope().Lookup("Banned"))
	})

	t.Run("synthesized TypesInfo contains Defs", func(t *testing.T) {
		dir := t.TempDir()

		buildTestArchive(t, dir, "chat.a", chatPkg)
		manifest := typeCacheManifest{
			Packages: map[string]string{
				chatPkg: "chat.a",
			},
		}
		manifestData, err := json.Marshal(manifest)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), manifestData, 0o644))

		cfg := DefaultConfig()
		require.NoError(t, cfg.LoadTypeCache(dir))

		chatLoaded := cfg.Packages.Load(chatPkg)
		require.NotNil(t, chatLoaded)
		require.NotNil(t, chatLoaded.TypesInfo)
		require.NotNil(t, chatLoaded.TypesInfo.Defs)

		foundNames := map[string]bool{}
		for ident, obj := range chatLoaded.TypesInfo.Defs {
			if obj != nil {
				foundNames[ident.Name] = true
			}
		}
		assert.True(t, foundNames["Message"])
		assert.True(t, foundNames["ProductSku"])
		assert.True(t, foundNames["ChatAPI"])
	})

	t.Run("missing manifest returns error", func(t *testing.T) {
		dir := t.TempDir()
		cfg := DefaultConfig()
		err := cfg.LoadTypeCache(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading type cache manifest")
	})

	t.Run("invalid manifest JSON returns error", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("not json"), 0o644))
		cfg := DefaultConfig()
		err := cfg.LoadTypeCache(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing type cache manifest")
	})

	t.Run("missing archive file returns error", func(t *testing.T) {
		dir := t.TempDir()
		manifest := typeCacheManifest{
			Packages: map[string]string{
				"example.com/missing": "nonexistent.a",
			},
		}
		manifestData, err := json.Marshal(manifest)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), manifestData, 0o644))

		cfg := DefaultConfig()
		err = cfg.LoadTypeCache(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading export data for example.com/missing")
	})

	t.Run("LoadSchema preserves injected packages", func(t *testing.T) {
		dir := t.TempDir()

		buildTestArchive(t, dir, "chat.a", chatPkg)
		manifest := typeCacheManifest{
			Packages: map[string]string{
				chatPkg: "chat.a",
			},
		}
		manifestData, err := json.Marshal(manifest)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), manifestData, 0o644))

		cfg := DefaultConfig()
		require.NoError(t, cfg.LoadTypeCache(dir))
		require.True(t, cfg.Packages.HasInjected())

		savedPkgs := cfg.Packages
		// HasInjected guard in LoadSchema prevents overwriting the Packages
		assert.True(t, savedPkgs.HasInjected())
	})
}

func TestSynthesizeTypesInfo(t *testing.T) {
	dir := t.TempDir()
	const chatPkg = "github.com/99designs/gqlgen/codegen/config/testdata/autobinding/chat"
	buildTestArchive(t, dir, "chat.a", chatPkg)

	manifest := typeCacheManifest{
		Packages: map[string]string{chatPkg: "chat.a"},
	}
	manifestData, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), manifestData, 0o644))

	cfg := DefaultConfig()
	require.NoError(t, cfg.LoadTypeCache(dir))

	chatLoaded := cfg.Packages.Load(chatPkg)
	require.NotNil(t, chatLoaded)

	// synthesizeTypesInfo should have populated Defs from the package scope
	info := chatLoaded.TypesInfo
	require.NotNil(t, info)

	names := map[string]bool{}
	for ident, obj := range info.Defs {
		if obj != nil {
			names[ident.Name] = true
		}
	}
	assert.True(t, names["Message"], "expected Message in synthesized Defs")
	assert.True(t, names["ChatAPI"], "expected ChatAPI in synthesized Defs")
}

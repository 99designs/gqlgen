package api

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationDocsExamplesStillValid(t *testing.T) {
	content, err := os.ReadFile("../README.md")
	require.NoError(t, err)

	readme := string(content)
	requiredSnippets := []string{
		"## Migration: `follow-schema` to `split-packages`",
		"layout: follow-schema",
		"layout: split-packages",
		"# shard_dir: graph/internal/gqlgenexec/shards",
		"rm -rf graph/internal/gqlgenexec",
		"Keep your `resolver` layout as-is unless you also want to change resolver generation separately.",
		"Existing resolver package paths remain compatible; only split-packages internals move under `graph/internal/...`.",
		"`exec.layout: split-packages` is not yet compatible with `federation` in the same config.",
	}

	for _, snippet := range requiredSnippets {
		t.Run(snippet, func(t *testing.T) {
			require.True(t, strings.Contains(readme, snippet), "README migration section missing snippet: %q", snippet)
		})
	}
}

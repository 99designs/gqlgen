package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitPackagesLargeFixtureExists(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	workDir := filepath.Join(wd, "testdata", "splitpackages_large")
	require.DirExists(t, workDir)
	require.FileExists(t, filepath.Join(workDir, "gqlgen.yml"))
	require.FileExists(t, filepath.Join(workDir, "graph", "schema.graphqls"))
}

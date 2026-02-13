package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitPackagesBenchHarnessSmoke(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	workDir := filepath.Join(wd, "testdata", "splitpackages")
	t.Cleanup(func() {
		cleanup(workDir)
	})

	payload, err := RunSplitPackagesBenchHarness(workDir)
	require.NoError(t, err)

	summary, err := ParseSplitPackagesBenchSummary(payload)
	require.NoError(t, err)
	require.Equal(t, splitPackagesBenchSummaryContractVersion, summary.ContractVersion)
	require.Equal(t, "split-packages", summary.Layout)
	require.Equal(t, "splitpackages", summary.Fixture)
	require.Positive(t, summary.GeneratedFileCount)
	require.Positive(t, summary.GeneratedBytes)
	require.GreaterOrEqual(t, summary.DurationMillis, int64(0))
}

func TestSplitPackagesBenchHarnessRelativeWorkDir(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	srcDir := filepath.Join(wd, "testdata", "splitpackages")
	baseDir := filepath.Join(wd, "testdata")
	workDir, err := os.MkdirTemp(baseDir, "relative-splitpackages-")
	require.NoError(t, err)
	workDirName := filepath.Base(workDir)
	require.NoError(t, copySplitBenchFixture(srcDir, workDir))
	cleanup(workDir)
	t.Cleanup(func() {
		cleanup(workDir)
		_ = os.RemoveAll(workDir)
	})

	t.Chdir(baseDir)

	payload, err := RunSplitPackagesBenchHarness(workDirName)
	require.NoError(t, err)

	summary, err := ParseSplitPackagesBenchSummary(payload)
	require.NoError(t, err)
	require.Equal(t, workDirName, summary.Fixture)
	require.Positive(t, summary.GeneratedFileCount)
	require.Positive(t, summary.GeneratedBytes)
}

func TestSplitPackagesBenchHarnessUsesExecOutputDirForStats(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	srcDir := filepath.Join(wd, "testdata", "splitpackages")
	workDir, err := os.MkdirTemp(filepath.Join(wd, "testdata"), "custom-output-")
	require.NoError(t, err)
	require.NoError(t, copySplitBenchFixture(srcDir, workDir))

	configPath := filepath.Join(workDir, "gqlgen.yml")
	configContents, err := os.ReadFile(configPath)
	require.NoError(t, err)

	updatedConfig := strings.Replace(string(configContents), "filename: graph/generated.go", "filename: generated/generated.go", 1)
	require.NotEqual(t, string(configContents), updatedConfig)
	require.NoError(t, os.WriteFile(configPath, []byte(updatedConfig), 0o644))

	cleanup(workDir)
	t.Cleanup(func() {
		cleanup(workDir)
		_ = os.RemoveAll(workDir)
	})

	payload, err := RunSplitPackagesBenchHarness(workDir)
	require.NoError(t, err)

	summary, err := ParseSplitPackagesBenchSummary(payload)
	require.NoError(t, err)
	require.Equal(t, filepath.Base(workDir), summary.Fixture)
	require.Positive(t, summary.GeneratedFileCount)
	require.Positive(t, summary.GeneratedBytes)
}

type splitBenchThreshold struct {
	maxDurationMillis int64
	minGeneratedFiles int
	minGeneratedBytes int64
}

func evaluateSplitBenchThreshold(summary SplitPackagesBenchSummary, threshold splitBenchThreshold) error {
	if summary.DurationMillis > threshold.maxDurationMillis {
		return fmt.Errorf("duration_ms %d exceeds max %d", summary.DurationMillis, threshold.maxDurationMillis)
	}

	if summary.GeneratedFileCount < threshold.minGeneratedFiles {
		return fmt.Errorf("generated_file_count %d below min %d", summary.GeneratedFileCount, threshold.minGeneratedFiles)
	}

	if summary.GeneratedBytes < threshold.minGeneratedBytes {
		return fmt.Errorf("generated_bytes %d below min %d", summary.GeneratedBytes, threshold.minGeneratedBytes)
	}

	return nil
}

func TestSplitBenchThresholdEvaluator(t *testing.T) {
	baseSummary := SplitPackagesBenchSummary{
		ContractVersion:    splitPackagesBenchSummaryContractVersion,
		Layout:             "split-packages",
		Fixture:            "splitpackages",
		GeneratedFileCount: 14,
		GeneratedBytes:     51200,
		DurationMillis:     320,
	}

	tests := []struct {
		name      string
		summary   SplitPackagesBenchSummary
		threshold splitBenchThreshold
		wantErr   string
	}{
		{
			name:    "passes when summary meets all thresholds",
			summary: baseSummary,
			threshold: splitBenchThreshold{
				maxDurationMillis: 320,
				minGeneratedFiles: 14,
				minGeneratedBytes: 51200,
			},
		},
		{
			name:    "fails when duration exceeds max",
			summary: baseSummary,
			threshold: splitBenchThreshold{
				maxDurationMillis: 319,
				minGeneratedFiles: 14,
				minGeneratedBytes: 51200,
			},
			wantErr: "duration_ms 320 exceeds max 319",
		},
		{
			name: "fails when generated files below min",
			summary: SplitPackagesBenchSummary{
				ContractVersion:    splitPackagesBenchSummaryContractVersion,
				Layout:             "split-packages",
				Fixture:            "splitpackages",
				GeneratedFileCount: 10,
				GeneratedBytes:     51200,
				DurationMillis:     320,
			},
			threshold: splitBenchThreshold{
				maxDurationMillis: 400,
				minGeneratedFiles: 11,
				minGeneratedBytes: 51200,
			},
			wantErr: "generated_file_count 10 below min 11",
		},
		{
			name: "fails when generated bytes below min",
			summary: SplitPackagesBenchSummary{
				ContractVersion:    splitPackagesBenchSummaryContractVersion,
				Layout:             "split-packages",
				Fixture:            "splitpackages",
				GeneratedFileCount: 14,
				GeneratedBytes:     4096,
				DurationMillis:     320,
			},
			threshold: splitBenchThreshold{
				maxDurationMillis: 400,
				minGeneratedFiles: 14,
				minGeneratedBytes: 5000,
			},
			wantErr: "generated_bytes 4096 below min 5000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := json.Marshal(tt.summary)
			require.NoError(t, err)

			parsed, err := ParseSplitPackagesBenchSummary(payload)
			require.NoError(t, err)

			err = evaluateSplitBenchThreshold(parsed, tt.threshold)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func copySplitBenchFixture(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		dstPath := filepath.Join(dstDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, contents, info.Mode().Perm())
	})
}

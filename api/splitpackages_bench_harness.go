package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/99designs/gqlgen/codegen/config"
)

const splitPackagesBenchSummaryContractVersion = "v1"

type SplitPackagesBenchSummary struct {
	ContractVersion    string `json:"contract_version"`
	Layout             string `json:"layout"`
	Fixture            string `json:"fixture"`
	GeneratedFileCount int    `json:"generated_file_count"`
	GeneratedBytes     int64  `json:"generated_bytes"`
	DurationMillis     int64  `json:"duration_ms"`
}

func RunSplitPackagesBenchHarness(workDir string) ([]byte, error) {
	if workDir == "" {
		return nil, errors.New("workDir is required")
	}

	start := time.Now()
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get wd: %w", err)
	}

	if err := os.Chdir(workDir); err != nil {
		return nil, fmt.Errorf("chdir workDir: %w", err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	cfg, err := config.LoadConfigFromDefaultLocations()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if cfg.Exec.Layout != "split-packages" {
		return nil, fmt.Errorf("unsupported exec layout %q", cfg.Exec.Layout)
	}

	if err := Generate(cfg); err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}

	fileCount, totalBytes, err := splitPackagesGeneratedStats(cfg.Exec.Dir())
	if err != nil {
		return nil, err
	}

	summary := SplitPackagesBenchSummary{
		ContractVersion:    splitPackagesBenchSummaryContractVersion,
		Layout:             string(cfg.Exec.Layout),
		Fixture:            filepath.Base(workDir),
		GeneratedFileCount: fileCount,
		GeneratedBytes:     totalBytes,
		DurationMillis:     time.Since(start).Milliseconds(),
	}

	if err := ValidateSplitPackagesBenchSummary(summary); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(summary)
	if err != nil {
		return nil, fmt.Errorf("marshal summary: %w", err)
	}

	return payload, nil
}

func ParseSplitPackagesBenchSummary(payload []byte) (SplitPackagesBenchSummary, error) {
	var summary SplitPackagesBenchSummary
	if err := json.Unmarshal(payload, &summary); err != nil {
		return SplitPackagesBenchSummary{}, fmt.Errorf("unmarshal summary: %w", err)
	}

	if err := ValidateSplitPackagesBenchSummary(summary); err != nil {
		return SplitPackagesBenchSummary{}, err
	}

	return summary, nil
}

func ValidateSplitPackagesBenchSummary(summary SplitPackagesBenchSummary) error {
	if summary.ContractVersion != splitPackagesBenchSummaryContractVersion {
		return fmt.Errorf("invalid contract version %q", summary.ContractVersion)
	}

	if summary.Layout != "split-packages" {
		return fmt.Errorf("invalid layout %q", summary.Layout)
	}

	if summary.Fixture == "" {
		return errors.New("fixture is required")
	}

	if summary.GeneratedFileCount <= 0 {
		return fmt.Errorf("generated_file_count must be > 0, got %d", summary.GeneratedFileCount)
	}

	if summary.GeneratedBytes <= 0 {
		return fmt.Errorf("generated_bytes must be > 0, got %d", summary.GeneratedBytes)
	}

	if summary.DurationMillis < 0 {
		return fmt.Errorf("duration_ms must be >= 0, got %d", summary.DurationMillis)
	}

	return nil
}

func splitPackagesGeneratedStats(generatedDir string) (int, int64, error) {
	fileCount := 0
	totalBytes := int64(0)

	err := filepath.WalkDir(generatedDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		name := filepath.Base(path)
		if name != "generated.go" && !strings.HasSuffix(name, ".generated.go") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		fileCount++
		totalBytes += info.Size()
		return nil
	})
	if err != nil {
		return 0, 0, fmt.Errorf("collect generated stats: %w", err)
	}

	return fileCount, totalBytes, nil
}

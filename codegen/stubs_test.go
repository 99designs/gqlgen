package codegen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestStubGenerator(t *testing.T) {
	updated := generateTestStubs("testdata/stubs/no_resolver.go", "App")

	assertFile(t, updated, "testdata/stubs/no_resolver.go", `
		package stubs

		type App struct{}

		func (r *App) Query() QueryResolver {
			return &QueryResolver{r}
		}

		type QueryResolver struct{ *App }

		func (r *App) User() UserResolver {
			return &UserResolver{r}
		}

		type UserResolver struct{ *App }
	`)
}

func assertFile(t *testing.T, updated map[string]string, name string, expected string) {
	actual, found := updated[abs(name)]
	if !found {
		t.Errorf("File %s not changed", name)
		return
	}
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(strings.TrimSpace(expected), strings.TrimSpace(actual), true)

	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual && strings.TrimSpace(d.Text) != "" {
			t.Errorf("file %s does not match expected output:\n%s", name, dmp.DiffPrettyText(diffs))
			return
		}
	}
}

func generateTestStubs(resolver string, rootType string) map[string]string {
	cfg := &Config{
		Exec: PackageConfig{
			Filename: "testdata/stubs/resolver.go",
		},
		Model: PackageConfig{
			Filename: "testdata/stubs/model.go",
		},
		Resolver: PackageConfig{
			Filename: resolver,
			Type:     rootType,
		},
	}

	err := cfg.Normalize()
	if err != nil {
		panic(err)
	}
	edits, err := getUpdatedStubs(cfg)
	if err != nil {
		fmt.Println(edits)
		panic(err)
	}
	return edits
}

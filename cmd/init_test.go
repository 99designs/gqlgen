package cmd_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/cmd"
	_ "github.com/99designs/gqlgen/graphql"
)

func TestWhenGoModMissing(t *testing.T) {
	dir := t.TempDir()
	os.Chdir(dir)
	err := cmd.DoInit("", "", "")
	if err != cmd.ErrGoModNotExist {
		t.Error("Expected ErrGoModNotExist")
	}
}

func doTestWhenFileExists(t *testing.T, filename string) {
	createNewDirForHappyPath(t)
	ioutil.WriteFile(filename, []byte{}, 0644)
	err := cmd.DoInit("gqlgen.yml", "server.go", "schema.graphqls")
	if err == nil {
		t.Error("Expected an error")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%s already exists", filename)) {
		t.Error("Expected an error")
	}
}

func TestWhenConfigAlreadyExists(t *testing.T) {
	doTestWhenFileExists(t, "gqlgen.yml")
}

func TestWhenServerAlreadyExists(t *testing.T) {
	doTestWhenFileExists(t, "server.go")
}

func TestWhenSchemaAlreadyExists(t *testing.T) {
	doTestWhenFileExists(t, "schema.graphqls")
}

func mustRun(name string, cmd ...string) {
	fmt.Println(name, cmd)
	runner := exec.Command(name, cmd...)
	out, err := runner.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		panic(err.Error())
	}
}

func createNewDirForHappyPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err.Error())
	}
	dir := t.TempDir()
	err = os.Chdir(dir)
	if err != nil {
		t.Fatal(err.Error())
	}
	ioutil.WriteFile("go.mod", []byte(fmt.Sprintf("module gqlgen.com/inittest\nreplace github.com/99designs/gqlgen => %s/..\n", cwd)), 0644)
	mustRun("go", "get", "-d", "github.com/99designs/gqlgen")
}

func TestHappyPath(t *testing.T) {
	createNewDirForHappyPath(t)
	mustRun("go", "run", "github.com/99designs/gqlgen", "init")
	files := findFiles()

	if !reflect.DeepEqual(files, []string{".", "go.mod", "go.sum", "gqlgen.yml", "graph", "graph/generated", "graph/generated/generated.go", "graph/model", "graph/model/models_gen.go", "graph/resolver.go", "graph/schema.graphqls", "graph/schema.resolvers.go", "server.go"}) {
		t.Errorf("Unexpected file list: %v", files)
	}
}

func findFiles() (filenames []string) {
	filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		filenames = append(filenames, path)
		return nil
	})
	return
}

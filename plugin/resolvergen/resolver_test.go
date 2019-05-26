package resolvergen

import (
	"fmt"
	"go/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/vektah/gqlparser/ast"
)

func TestPlugin_Name(t *testing.T) {
	t.Run("test plugin name", func(t *testing.T) {
		m := &Plugin{}
		if got, want := m.Name(), "resovlergen"; got != want {
			t.Errorf("Plugin.Name() = %v, want %v", got, want)
		}
	})
}

// Types for testing code generation, both Mutation and
// MutationResolver must implement types.Type.
type Mutation struct{}

func (m *Mutation) Underlying() types.Type {
	return m
}

func (m *Mutation) String() string {
	return "Mutation"
}

type MutationResolver struct{}

func (m *MutationResolver) Underlying() types.Type {
	return m
}

func (m *MutationResolver) String() string {
	return "MutationResolver"
}

func TestPlugin_GenerateCode(t *testing.T) {
	makeData := func(cfg config.PackageConfig) *codegen.Data {
		m := &Mutation{}
		obj := &codegen.Object{
			Definition: &ast.Definition{
				Name: fmt.Sprint(m),
			},
			Root: true,
			Fields: []*codegen.Field{
				&codegen.Field{
					IsResolver:  true,
					GoFieldName: "Name",
					TypeReference: &config.TypeReference{
						GO: m,
					},
				},
			},
			ResolverInterface: &MutationResolver{},
		}
		obj.Fields[0].Object = obj
		return &codegen.Data{
			Config: &config.Config{
				Resolver: cfg,
			},
			Objects: codegen.Objects{obj},
		}
	}

	t.Run("renders expected contents", func(t *testing.T) {
		m := &Plugin{}

		// use a temp dir to ensure generated file uniqueness,
		// since if a file already exists it won't be
		// overwritten.
		tempDir, err := ioutil.TempDir("", "resolvergen-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)
		filename := filepath.Join(tempDir, "generated.go")

		data := makeData(config.PackageConfig{
			Filename: filename,
			Package:  "customresolver",
			Type:     "CustomResolverType",
		})
		if err := m.GenerateCode(data); err != nil {
			t.Fatal(err)
		}

		byteContents, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		contents := string(byteContents)

		want := "package customresolver"
		if !strings.Contains(contents, want) {
			t.Fatalf("expected package name not found: want = %q\n%s", want, contents)
		}

		// Skip all white-space chars after start and want
		// length. Useful to jump to next non-white character
		// contents for generated code assertions.
		skipWhitespace := func(start int, want string) int {
			return start + len(want) + strings.IndexFunc(
				string(contents[start+len(want):]),
				func(r rune) bool { return !unicode.IsSpace(r) },
			)
		}
		// Check if want begins at the given start point.
		lookingAt := func(start int, want string) bool {
			return strings.Index(string(contents[start:]), want) == 0
		}

		// Assert Mutation method contents for *CustomResolverType
		want = "func (r *CustomResolverType) Mutation() MutationResolver {"
		start := strings.Index(contents, want)
		if start == -1 {
			t.Fatalf("mutation method for custom resolver not found: want = %q\n%s", want, contents)
		}
		start = skipWhitespace(start, want)
		want = "return &mutationCustomResolverType{r}"
		if !lookingAt(start, want) {
			t.Fatalf("unexpected return on mutation method for custom resolver: want = %q\n%s", want, contents)
		}
		start = skipWhitespace(start, want)
		want = "}"
		if !lookingAt(start, want) {
			t.Fatalf("unexpected contents on mutation method for custom resolver: want = %q\n%s", want, contents)
		}

		want = "type mutationCustomResolverType struct{ *CustomResolverType }"
		if !strings.Contains(contents, want) {
			t.Fatalf("expected embedded resolver type struct not found: want = %q\n%s", want, contents)
		}

		// Assert Name method contents for *mutationCustomResolverType
		want = "func (r *mutationCustomResolverType) Name(ctx context.Context) (Mutation, error) {"
		start = strings.Index(contents, want)
		if start == -1 {
			t.Fatalf("Name method for mutation custom resolver type not found: want = %q\n%s", want, contents)
		}
		start = skipWhitespace(start, want)
		want = `panic("not implemented")`
		if !lookingAt(start, want) {
			t.Fatalf("unexpected Name method contents for mutation custom resolver type: want = %q\n%s", want, contents)
		}
		start = skipWhitespace(start, want)
		want = "}"
		if !lookingAt(start, want) {
			t.Fatalf("unexpected Name method contents for mutation custom resolver type: want = %q\n%s", want, contents)
		}
	})
}

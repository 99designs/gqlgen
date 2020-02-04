package resolvergen

import (
	"fmt"
	"io/ioutil"
	"syscall"
	"testing"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestLayoutSingleFile(t *testing.T) {
	_ = syscall.Unlink("testdata/out/singlefile/resolver.go")

	cfg, err := config.LoadConfig("testdata/singlefile/gqlgen.yml")
	require.NoError(t, err)
	p := Plugin{}

	require.NoError(t, cfg.Init())

	data, err := codegen.BuildData(cfg)
	if err != nil {
		panic(err)
	}

	require.NoError(t, p.GenerateCode(data))
	assertNoErrors(t, "github.com/99designs/gqlgen/plugin/resolvergen/testdata/singlefile/out")
}

func TestLayoutFollowSchema(t *testing.T) {
	_ = syscall.Unlink("testdata/out/followschema/resolver.go")

	cfg, err := config.LoadConfig("testdata/followschema/gqlgen.yml")
	require.NoError(t, err)
	p := Plugin{}

	require.NoError(t, cfg.Init())

	data, err := codegen.BuildData(cfg)
	if err != nil {
		panic(err)
	}

	require.NoError(t, p.GenerateCode(data))
	assertNoErrors(t, "github.com/99designs/gqlgen/plugin/resolvergen/testdata/followschema/out")

	b, err := ioutil.ReadFile("testdata/followschema/out/schema.resolvers.go")
	require.NoError(t, err)
	source := string(b)

	require.Contains(t, source, "// CustomerResolverType.Resolver implementation")
	require.Contains(t, source, "// CustomerResolverType.Name implementation")
	require.Contains(t, source, "// AUserHelperFunction implementation")
}

func assertNoErrors(t *testing.T, pkg string) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedTypes |
			packages.NeedTypesSizes}, pkg)
	if err != nil {
		panic(err)
	}

	hasErrors := false
	for _, pkg := range pkgs {
		for _, err := range pkg.Errors {
			hasErrors = true
			fmt.Println(err.Pos + ":" + err.Msg)
		}
	}
	if hasErrors {
		t.Fatal("see compilation errors above")
	}
}

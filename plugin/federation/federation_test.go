package federation

import (
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

func TestInjectSources(t *testing.T) {
	cfg, err := config.LoadConfig("test_data/gqlgen.yml")
	require.NoError(t, err)
	f := &federation{}
	f.InjectSources(cfg)
	if len(cfg.AdditionalSources) != 2 {
		t.Fatalf("expected 2 additional sources but got %v", len(cfg.AdditionalSources))
	}
}

func TestMutateSchema(t *testing.T) {
	f := &federation{}

	schema, gqlErr := gqlparser.LoadSchema(&ast.Source{
		Name: "schema.graphql",
		Input: `type Query {
			hello: String!
			world: String!
		}`,
	})
	if gqlErr != nil {
		t.Fatal(gqlErr)
	}

	err := f.MutateSchema(schema)
	require.NoError(t, err)
}

func TestGetSDL(t *testing.T) {
	cfg, err := config.LoadConfig("test_data/gqlgen.yml")
	require.NoError(t, err)
	f := &federation{}
	_, err = f.getSDL(cfg)
	require.NoError(t, err)
}

func TestMutateConfig(t *testing.T) {
	cfg, err := config.LoadConfig("test_data/gqlgen.yml")
	require.NoError(t, err)

	f := &federation{}
	f.InjectSources(cfg)

	require.NoError(t, cfg.LoadSchema())
	require.NoError(t, f.MutateSchema(cfg.Schema))
	require.NoError(t, cfg.Init())
	require.NoError(t, f.MutateConfig(cfg))

}

func TestInjectSourcesNoKey(t *testing.T) {
	cfg, err := config.LoadConfig("test_data/nokey.yml")
	require.NoError(t, err)
	f := &federation{}
	f.InjectSources(cfg)
	if len(cfg.AdditionalSources) != 1 {
		t.Fatalf("expected an additional source but got %v", len(cfg.AdditionalSources))
	}
}

func TestGetSDLNoKey(t *testing.T) {
	cfg, err := config.LoadConfig("test_data/nokey.yml")
	require.NoError(t, err)
	f := &federation{}
	_, err = f.getSDL(cfg)
	require.NoError(t, err)
}

func TestMutateConfigNoKey(t *testing.T) {
	cfg, err := config.LoadConfig("test_data/nokey.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.Init())

	f := &federation{}
	err = f.MutateConfig(cfg)
	require.NoError(t, err)
}

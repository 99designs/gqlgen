package federation

import (
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

func TestInjectSources(t *testing.T) {
	var cfg config.Config
	f := &federation{}
	f.InjectSources(&cfg)
	if len(cfg.AdditionalSources) != 2 {
		t.Fatalf("expected an additional source but got %v", len(cfg.AdditionalSources))
	}
}

func TestMutateSchema(t *testing.T) {
	f := &federation{}

	schema, gqlErr := gqlparser.LoadSchema(&ast.Source{
		Name: "schema.graphql",
		Input: `type Query {
			hello: String!
		}`,
	})
	if gqlErr != nil {
		t.Fatal(gqlErr)
	}
	err := f.MutateSchema(schema)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetSDL(t *testing.T) {
	cfg, err := config.LoadConfig("test_data/gqlgen.yml")
	require.NoError(t, err)
	f := &federation{}
	_, err = f.getSDL(cfg)
	require.NoError(t, err)
}

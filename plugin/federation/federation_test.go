package federation_test

import (
	"testing"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/federation"
	"github.com/vektah/gqlparser"
	"github.com/vektah/gqlparser/ast"
)

func TestInjectSources(t *testing.T) {
	var cfg config.Config
	f := federation.New().(plugin.SourcesInjector)
	f.InjectSources(&cfg)
	if len(cfg.AdditionalSources) != 2 {
		t.Fatalf("expected an additional source but got %v", len(cfg.AdditionalSources))
	}
}

func TestMutateSchema(t *testing.T) {
	f := federation.New()

	schema, gqlErr := gqlparser.LoadSchema(&ast.Source{
		Name: "schema.graphql",
		Input: `type Query {
			hello: String!
		}`,
	})
	if gqlErr != nil {
		t.Fatal(gqlErr)
	}
	err := f.(codegen.SchemaMutator).MutateSchema(schema)
	if err != nil {
		t.Fatal(err)
	}
}

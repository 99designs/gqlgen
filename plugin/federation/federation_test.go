package federation

import (
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/require"
)

func TestWithEntities(t *testing.T) {
	f, cfg := load(t, "test_data/gqlgen.yml")

	require.Equal(t, []string{"ExternalExtension", "Hello", "MoreNesting", "NestedKey", "VeryNestedKey", "World"}, cfg.Schema.Types["_Entity"].Types)

	require.Len(t, cfg.Schema.Types["Entity"].Fields, 5)

	require.Equal(t, "findExternalExtensionByUpc", cfg.Schema.Types["Entity"].Fields[0].Name)
	require.Equal(t, "findHelloByName", cfg.Schema.Types["Entity"].Fields[1].Name)
	require.Equal(t, "findNestedKeyByIDAndHelloName", cfg.Schema.Types["Entity"].Fields[2].Name)
	require.Equal(t, "findVeryNestedKeyByIDAndHelloNameAndWorldFooAndWorldBarAndMoreWorldFoo", cfg.Schema.Types["Entity"].Fields[3].Name)
	require.Equal(t, "findWorldByFooAndBar", cfg.Schema.Types["Entity"].Fields[4].Name)

	require.NoError(t, f.MutateConfig(cfg))

	require.Equal(t, "ExternalExtension", f.Entities[0].Name)
	require.Len(t, f.Entities[0].KeyFields, 1)
	require.Equal(t, "upc", f.Entities[0].KeyFields[0].Definition.Name)
	require.Equal(t, "String", f.Entities[0].KeyFields[0].Definition.Type.Name())

	require.Equal(t, "Hello", f.Entities[1].Name)
	require.Len(t, f.Entities[1].KeyFields, 1)
	require.Equal(t, "name", f.Entities[1].KeyFields[0].Definition.Name)
	require.Equal(t, "String", f.Entities[1].KeyFields[0].Definition.Type.Name())

	require.Equal(t, "MoreNesting", f.Entities[2].Name)
	require.Len(t, f.Entities[2].KeyFields, 1)
	require.Equal(t, "id", f.Entities[2].KeyFields[0].Definition.Name)
	require.Equal(t, "String", f.Entities[2].KeyFields[0].Definition.Type.Name())

	require.Equal(t, "NestedKey", f.Entities[3].Name)
	require.Len(t, f.Entities[3].KeyFields, 2)
	require.Equal(t, "id", f.Entities[3].KeyFields[0].Definition.Name)
	require.Equal(t, "String", f.Entities[3].KeyFields[0].Definition.Type.Name())
	require.Equal(t, "helloName", f.Entities[3].KeyFields[1].Definition.Name)
	require.Equal(t, "String", f.Entities[3].KeyFields[1].Definition.Type.Name())

	require.Equal(t, "VeryNestedKey", f.Entities[4].Name)
	require.Len(t, f.Entities[4].KeyFields, 5)
	require.Equal(t, "id", f.Entities[4].KeyFields[0].Definition.Name)
	require.Equal(t, "String", f.Entities[4].KeyFields[0].Definition.Type.Name())
	require.Equal(t, "helloName", f.Entities[4].KeyFields[1].Definition.Name)
	require.Equal(t, "String", f.Entities[4].KeyFields[1].Definition.Type.Name())
	require.Equal(t, "worldFoo", f.Entities[4].KeyFields[2].Definition.Name)
	require.Equal(t, "String", f.Entities[4].KeyFields[2].Definition.Type.Name())
	require.Equal(t, "worldBar", f.Entities[4].KeyFields[3].Definition.Name)
	require.Equal(t, "Int", f.Entities[4].KeyFields[3].Definition.Type.Name())
	require.Equal(t, "moreWorldFoo", f.Entities[4].KeyFields[4].Definition.Name)
	require.Equal(t, "String", f.Entities[4].KeyFields[4].Definition.Type.Name())

	require.Len(t, f.Entities[4].Requires, 2)
	require.Equal(t, f.Entities[4].Requires[0].Name, "id")
	require.Equal(t, f.Entities[4].Requires[1].Name, "helloSecondary")

	require.Equal(t, "World", f.Entities[5].Name)
	require.Len(t, f.Entities[5].KeyFields, 2)
	require.Equal(t, "foo", f.Entities[5].KeyFields[0].Definition.Name)
	require.Equal(t, "String", f.Entities[5].KeyFields[0].Definition.Type.Name())
	require.Equal(t, "bar", f.Entities[5].KeyFields[1].Definition.Name)
	require.Equal(t, "Int", f.Entities[5].KeyFields[1].Definition.Type.Name())
}

func TestNoEntities(t *testing.T) {
	f, cfg := load(t, "test_data/nokey.yml")

	err := f.MutateConfig(cfg)
	require.NoError(t, err)
}

func load(t *testing.T, name string) (*federation, *config.Config) {
	t.Helper()

	cfg, err := config.LoadConfig(name)
	require.NoError(t, err)

	f := &federation{}
	cfg.Sources = append(cfg.Sources, f.InjectSourceEarly())
	require.NoError(t, cfg.LoadSchema())

	if src := f.InjectSourceLate(cfg.Schema); src != nil {
		cfg.Sources = append(cfg.Sources, src)
	}
	require.NoError(t, cfg.LoadSchema())

	require.NoError(t, cfg.Init())
	return f, cfg
}

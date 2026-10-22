package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/codegen/config"
)

// A GraphQL field bound to a Go struct field silently ignores its arguments.
// Introspection fields with arguments (fields, enumValues, inputFields and args,
// each taking includeDeprecated) must therefore bind to methods that receive
// them, and __schema/__type must keep gqlparser's definitions.
func TestIntrospectionFieldsReceiveTheirArguments(t *testing.T) {
	cfg, err := config.LoadConfig("testdata/argbinding/gqlgen.yml")
	require.NoError(t, err)
	require.NoError(t, cfg.LoadSchema())
	require.NoError(t, cfg.Init())

	data, err := BuildData(cfg)
	require.NoError(t, err)

	checked := 0
	for _, obj := range data.Objects {
		if !obj.IsReserved() {
			continue
		}
		for _, f := range obj.Fields {
			if len(f.Args) == 0 {
				continue
			}
			checked++
			require.True(t, f.IsMethod(),
				"%s.%s takes arguments but binds to %s, which cannot receive them",
				obj.Name, f.Name, f.GoFieldName)
		}
	}
	require.NotZero(t, checked, "no introspection field with arguments was found")

	t.Run("introspection roots use gqlparser's definitions", func(t *testing.T) {
		built := map[string]*Field{}
		for _, f := range data.Objects.ByName(cfg.Schema.Query.Name).Fields {
			built[f.Name] = f
		}
		for _, name := range []string{"__schema", "__type"} {
			require.Contains(t, built, name)
			require.Same(t, cfg.Schema.Query.Fields.ForName(name), built[name].FieldDefinition)
		}
		require.True(t, built["__schema"].TypeReference.IsPtr(),
			"IntrospectSchema returns *introspection.Schema although __schema is non-null")
	})
}

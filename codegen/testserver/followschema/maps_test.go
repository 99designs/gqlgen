package followschema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
)

func TestMaps(t *testing.T) {
	resolver := &Stub{}
	resolver.QueryResolver.MapStringInterface = func(ctx context.Context, in map[string]interface{}) (i map[string]interface{}, e error) {
		validateMapItemsType(t, in)
		return in, nil
	}
	resolver.QueryResolver.MapNestedStringInterface = func(ctx context.Context, in *NestedMapInput) (i map[string]interface{}, e error) {
		if in == nil {
			return nil, nil
		}
		validateMapItemsType(t, in.Map)
		return in.Map, nil
	}

	c := client.New(handler.NewDefaultServer(
		NewExecutableSchema(Config{Resolvers: resolver}),
	))
	t.Run("unset", func(t *testing.T) {
		var resp struct {
			MapStringInterface map[string]interface{}
		}
		err := c.Post(`query { mapStringInterface { a, b, c, nested { value } } }`, &resp)
		require.NoError(t, err)
		require.Nil(t, resp.MapStringInterface)
	})

	t.Run("nil", func(t *testing.T) {
		var resp struct {
			MapStringInterface map[string]interface{}
		}
		err := c.Post(`query { mapStringInterface(in: null) { a, b, c, nested { value } } }`, &resp)
		require.NoError(t, err)
		require.Nil(t, resp.MapStringInterface)
	})

	t.Run("values", func(t *testing.T) {
		var resp struct {
			MapStringInterface map[string]interface{}
		}
		err := c.Post(`query($value: CustomScalar!) { mapStringInterface(in: { a: "a", b: null, c: 42, nested: { value: $value } }) { a, b, c, nested { value } } }`, &resp, client.Var("value", "17"))
		require.NoError(t, err)
		require.Equal(t, "a", resp.MapStringInterface["a"])
		require.Nil(t, resp.MapStringInterface["b"])
		require.Equal(t, "42", resp.MapStringInterface["c"])
		require.NotNil(t, resp.MapStringInterface["nested"])
		require.IsType(t, map[string]interface{}{}, resp.MapStringInterface["nested"])
		require.Equal(t, "17", (resp.MapStringInterface["nested"].(map[string]interface{}))["value"])
	})

	t.Run("nested", func(t *testing.T) {
		var resp struct {
			MapNestedStringInterface map[string]interface{}
		}
		err := c.Post(`query { mapNestedStringInterface(in: { map: { a: "a", c: "42", nested: { value: 31 } } }) { a, b, c, nested { value } } }`, &resp)
		require.NoError(t, err)
		require.Equal(t, "a", resp.MapNestedStringInterface["a"])
		require.Nil(t, resp.MapNestedStringInterface["b"])
		require.Equal(t, "42", resp.MapNestedStringInterface["c"])
		require.NotNil(t, resp.MapNestedStringInterface["nested"])
		require.IsType(t, map[string]interface{}{}, resp.MapNestedStringInterface["nested"])
		require.Equal(t, "31", (resp.MapNestedStringInterface["nested"].(map[string]interface{}))["value"])
	})

	t.Run("nested nil", func(t *testing.T) {
		var resp struct {
			MapNestedStringInterface map[string]interface{}
		}
		err := c.Post(`query { mapNestedStringInterface(in: { map: null }) { a, b, c, nested { value } } }`, &resp)
		require.NoError(t, err)
		require.Nil(t, resp.MapNestedStringInterface)
	})
}

func validateMapItemsType(t *testing.T, in map[string]interface{}) {
	for k, v := range in {
		switch k {
		case "a":
			require.IsType(t, "", v)
		case "b":
			require.IsType(t, new(int), v)
		case "c":
			require.IsType(t, new(CustomScalar), v)
		case "nested":
			require.IsType(t, new(MapNested), v)
		default:
			require.Failf(t, "unexpected key in map", "key %q was not expected in map", k)
		}
	}
}

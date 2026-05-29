package graphql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFederationRequiresForBatch(t *testing.T) {
	representations := []map[string]any{
		{"__typename": "Product", "id": "1", "variations": []any{map[string]any{"price": 10}}},
		{"__typename": "Product", "id": "2", "variations": []any{map[string]any{"price": 20}}},
	}

	entitiesFC := &FieldContext{
		Object: "Query",
		Args:   map[string]any{"representations": representations},
	}
	entitiesListIdx := 0
	entitiesFCWithIndex := &FieldContext{
		Object: "Product",
		Index:  &entitiesListIdx,
	}
	sizeFC := &FieldContext{
		Object: "Product",
	}

	ctx := context.Background()
	ctx = WithFieldContext(ctx, entitiesFC)
	ctx = WithFieldContext(ctx, entitiesFCWithIndex)
	ctx = WithFieldContext(ctx, sizeFC)

	t.Run("single parent uses entity index", func(t *testing.T) {
		got, err := FederationRequiresForBatch(ctx, 1, nil)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, representations[0], got[0])
	})

	t.Run("batch with index map", func(t *testing.T) {
		got, err := FederationRequiresForBatch(ctx, 2, map[int]int{0: 0, 1: 1})
		require.NoError(t, err)
		require.Equal(t, representations[0], got[0])
		require.Equal(t, representations[1], got[1])
	})

	t.Run("batch with remapped index", func(t *testing.T) {
		reps := make([]map[string]any, 6)
		copy(reps, representations)
		reps[2] = map[string]any{"__typename": "Product", "id": "3"}
		reps[5] = map[string]any{"__typename": "Product", "id": "4"}
		entitiesFC.Args["representations"] = reps

		got, err := FederationRequiresForBatch(ctx, 2, map[int]int{2: 0, 5: 1})
		require.NoError(t, err)
		require.Equal(t, reps[2], got[0])
		require.Equal(t, reps[5], got[1])
	})
}

func TestFederationRequiresForBatch_nestedParentChain(t *testing.T) {
	representations := []map[string]any{
		{"__typename": "Other"},
		{"__typename": "Other"},
		{"__typename": "Product", "id": "1", "variations": []any{map[string]any{"price": 10}}},
	}

	entitiesFC := &FieldContext{
		Object: "Query",
		Args:   map[string]any{"representations": representations},
	}
	entityIdx := 2
	entityFC := &FieldContext{
		Object: "Product",
		Index:  &entityIdx,
	}
	pricingFC := &FieldContext{
		Object: "Pricing",
	}
	bestPriceFC := &FieldContext{
		Object: "Pricing",
	}

	ctx := context.Background()
	ctx = WithFieldContext(ctx, entitiesFC)
	ctx = WithFieldContext(ctx, entityFC)
	ctx = WithFieldContext(ctx, pricingFC)
	ctx = WithFieldContext(ctx, bestPriceFC)

	t.Run("finds representations through intermediate parent", func(t *testing.T) {
		got, err := FederationRequiresForBatch(ctx, 1, nil)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, representations[2], got[0])
	})
}

func TestFindFederationRepresentations(t *testing.T) {
	reps := []map[string]any{{"id": "1"}}
	entitiesFC := &FieldContext{
		Args: map[string]any{"representations": reps},
	}

	t.Run("direct ancestor", func(t *testing.T) {
		fc := &FieldContext{Object: "Product"}
		ctx := WithFieldContext(WithFieldContext(context.Background(), entitiesFC), fc)
		got, err := findFederationRepresentations(GetFieldContext(ctx))
		require.NoError(t, err)
		require.Equal(t, reps, got)
	})

	t.Run("missing", func(t *testing.T) {
		_, err := findFederationRepresentations(&FieldContext{Object: "Query"})
		require.ErrorIs(t, err, errNotWithinEntities)
	})
}

func TestFindEntityRepresentationIndex(t *testing.T) {
	idx := 3
	entityFC := &FieldContext{Object: "Product", Index: &idx}
	pricingFC := &FieldContext{Object: "Pricing"}
	fieldFC := &FieldContext{Object: "Pricing"}

	ctx := context.Background()
	ctx = WithFieldContext(ctx, entityFC)
	ctx = WithFieldContext(ctx, pricingFC)
	ctx = WithFieldContext(ctx, fieldFC)

	got, ok := findEntityRepresentationIndex(GetFieldContext(ctx))
	require.True(t, ok)
	require.Equal(t, 3, got)
}

func TestFederationRequiresForBatch_outsideEntities(t *testing.T) {
	_, err := FederationRequiresForBatch(context.Background(), 1, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, errNotWithinEntities)
}

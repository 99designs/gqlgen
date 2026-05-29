package graphql

import (
	"context"
	"errors"
	"fmt"
)

// FederationRequiresForBatch builds per-parent @requires payloads for batch resolvers
// when computed_requires is enabled. The returned slice has length count and aligns with
// the objs/parents slice passed to the batch resolver.
//
// Not used with explicit_requires: required fields are populated on each entity via
// Populate*Requires before field resolution, so batch resolvers read objs[i] directly.
//
// It is called from generated resolveBatch_* code while resolving a field that has both
// @requires and @goField(batch: true), not while parsing the _federationRequires argument.
// GetFieldContext(ctx) is therefore the @requires field (e.g. Product.bestPrice), not the
// argument directive context.
//
// FieldContext ancestry (each WithFieldContext links Parent to the previous context):
//
//	Query._entities          Args["representations"] = input list from the router
//	  └─ entity[i]           Index = i (set when marshaling the _entities array)
//	       └─ @requires field   current fc when FederationRequiresForBatch runs
//
// Example query:
//
//	query ($representations: [_Any!]!) {
//	  _entities(representations: $representations) {
//	    ... on Product { bestPrice }
//	  }
//	}
//
// We walk Parent to find the _entities args and the entity Index instead of assuming a
// fixed depth: the standard path is two parents (field → entity → _entities), but an extra
// intermediate object context between the entity and the @requires field is also supported.
//
// Limitations (same contract as populateFromRepresentations in the federation plugin):
//   - Must run under _entities; there is no representations arg elsewhere.
//   - Intended for @requires on the federated entity type (or its selection tree under
//     that entity's context), not for arbitrary non-entity resolvers.
//
// See plugin/federation/constants.go (populateFromRepresentations) for the non-batch path.
func FederationRequiresForBatch(
	ctx context.Context,
	count int,
	indexMap map[int]int,
) ([]map[string]any, error) {
	// Full list passed to _entities(representations: ...); each map is one router representation.
	representations, err := federationRepresentations(ctx)
	if err != nil {
		return nil, err
	}

	// One requires payload per batch parent: requires[i] is passed to the resolver for objs[i].
	// Non-batch computed_requires uses a single map; batch must not reuse one map for all parents.
	requires := make([]map[string]any, count)
	for batchIdx := range count {
		// batchIdx is the position in the grouped parents slice (0 .. count-1).
		// repIdx is the index into representations for that parent's federation payload.
		//
		// Mapping depends on how parents were batched:
		//   - indexMap set (interface/union lists): keys are original list indices, values are batch positions.
		//   - count == 1: use the entity's Index from FieldContext (the _entities array slot).
		//   - otherwise: batchIdx == repIdx (homogeneous slice aligned with representations order).
		repIdx, err := federationRepresentationIndex(ctx, batchIdx, count, indexMap)
		if err != nil {
			return nil, err
		}
		if repIdx < 0 || repIdx >= len(representations) {
			return nil, fmt.Errorf("representation not found for batch index %d", batchIdx)
		}
		// Same map populateFromRepresentations would return for this entity alone.
		requires[batchIdx] = representations[repIdx]
	}
	return requires, nil
}

var errNotWithinEntities = errors.New("must be called from within _entities")

func federationRepresentations(ctx context.Context) ([]map[string]any, error) {
	fc := GetFieldContext(ctx)
	if fc == nil {
		return nil, errNotWithinEntities
	}
	return findFederationRepresentations(fc)
}

// findFederationRepresentations returns the _entities representations argument from the
// nearest ancestor FieldContext that carries it (typically Query._entities).
func findFederationRepresentations(fc *FieldContext) ([]map[string]any, error) {
	for it := fc; it != nil; it = it.Parent {
		if it.Args == nil {
			continue
		}
		representations, ok := it.Args["representations"].([]map[string]any)
		if ok {
			return representations, nil
		}
	}
	return nil, errNotWithinEntities
}

// findEntityRepresentationIndex returns the index of the current entity in the
// representations slice (the ancestor FieldContext with Index set, usually the entity
// node directly under _entities).
func findEntityRepresentationIndex(fc *FieldContext) (int, bool) {
	for it := fc; it != nil; it = it.Parent {
		if it.Index != nil {
			return *it.Index, true
		}
	}
	return 0, false
}

func federationRepresentationIndex(
	ctx context.Context,
	batchIdx int,
	count int,
	indexMap map[int]int,
) (int, error) {
	if indexMap != nil {
		for orig, mapped := range indexMap {
			if mapped == batchIdx {
				return orig, nil
			}
		}
		return 0, fmt.Errorf("batch index %d not found in index map", batchIdx)
	}
	if count == 1 {
		fc := GetFieldContext(ctx)
		if fc != nil {
			if idx, ok := findEntityRepresentationIndex(fc); ok {
				return idx, nil
			}
		}
	}
	return batchIdx, nil
}

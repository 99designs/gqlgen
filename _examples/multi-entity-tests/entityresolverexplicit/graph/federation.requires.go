package graph

import (
	"context"
	"encoding/json"

	"entityresolverexplicit/graph/model"
)

// PopulateProductRequires is the requires populator for the Product entity.
func (ec *executionContext) PopulateProductRequires(ctx context.Context, entity *model.Product, reps map[string]any) error {
	b, _ := json.Marshal(reps)
	println(string(b))
	json.Unmarshal(b, entity)
	return nil
}

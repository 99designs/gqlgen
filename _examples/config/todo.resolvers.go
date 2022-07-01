package config

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
)

// ID is the resolver for the id field.
func (r *todoResolver) ID(ctx context.Context, obj *Todo) (string, error) {
	if obj.ID != "" {
		return obj.ID, nil
	}

	obj.ID = fmt.Sprintf("TODO:%d", obj.DatabaseID)

	return obj.ID, nil
}

// Todo returns TodoResolver implementation.
func (r *Resolver) Todo() TodoResolver { return &todoResolver{r} }

type todoResolver struct{ *Resolver }

package graph

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/_examples/federation/reviews/graph/model"
)

// PopulateUserRequires is the requires populator for the User entity.
func (ec *executionContext) PopulateUserRequires(ctx context.Context, entity *model.User, reps map[string]interface{}) error {
	panic(fmt.Errorf("not implemented: PopulateUserRequires"))
}

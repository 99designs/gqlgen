// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
package graph

import "github.com/99designs/gqlgen/_examples/federation/accounts/graph/model"

type Resolver struct{}

func (r *Resolver) HostForUserID(id string) (*model.EmailHost, error) {
	return &model.EmailHost{
		ID:   id,
		Name: "Email Host " + id,
	}, nil
}

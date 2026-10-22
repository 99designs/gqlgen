package graph

import "github.com/99designs/gqlgen/_examples/federation/v1/accounts/graph/model"

// Resolver serves a fixed set of users.
type Resolver struct{}

func userByID(id string) *model.User {
	username := "user-" + id
	if id == "1" {
		username = "sal"
	}
	return &model.User{ID: id, Username: username}
}

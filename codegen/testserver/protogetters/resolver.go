package protogetters

import "github.com/99designs/gqlgen/codegen/testserver/protogetters/models"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	users map[string]*models.User
}

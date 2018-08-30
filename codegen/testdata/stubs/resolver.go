package stubs

import "context"

type ResolverRoot interface {
	Query() QueryResolver
	User() UserResolver
}

type QueryResolver interface {
	User(ctx context.Context, id int) (*User, error)
}

type UserResolver interface {
	Friends(ctx context.Context) ([]*User, error)
}

type User struct {
	Name string
}

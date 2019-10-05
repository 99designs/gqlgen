//go:generate go run ./generate/generate.go
package introspection

import "context"

type Root struct {
}

func (r *Root) Mutation() MutationResolver {
	return &mutationResolver{}
}

func (r *Root) Query() QueryResolver {
	return &queryResolver{}
}

type mutationResolver struct {
}

func (m *mutationResolver) CreateUser(ctx context.Context, input *UserCreateInput) (*User, error) {
	return &User{
		ID:           "1",
		Email:        input.Email,
		PasswordHash: "212f6d7f6885bc4acea7",
	}, nil
}

type queryResolver struct {
}

func (q *queryResolver) CurrentUser(ctx context.Context) (*User, error) {
	return &User{
		ID:           "1",
		Email:        "example@example.com",
		PasswordHash: "212f6d7f6885bc4acea7",
	}, nil
}

//go:generate go run ../../testdata/gqlgen.go

package config

func New() Config {
	c := Config{
		Resolvers: &Resolver{
			todos: []*Todo{
				{DatabaseID: 1, Description: "A todo not to forget", Done: false},
				{DatabaseID: 2, Description: "This is the most important", Done: false},
				{DatabaseID: 3, Description: "Please do this or else", Done: false},
			},
			nextID: 3,
		},
	}
	return c
}

type Resolver struct {
	todos  []*Todo
	nextID int
}

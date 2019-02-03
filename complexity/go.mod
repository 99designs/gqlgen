module github.com/99designs/gqlgen/complexity

// Rewrite
require github.com/99designs/gqlgen/graphql v0.4.5-0.20190203203210-e4679b668de0

replace github.com/99designs/gqlgen/graphql => ../graphql

require (
	github.com/stretchr/testify v1.3.0
	// Actual dependencies
	github.com/vektah/gqlparser v1.1.0
)

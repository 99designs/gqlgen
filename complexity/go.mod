module github.com/99designs/gqlgen/complexity

// Rewrite
require github.com/99designs/gqlgen/graphql v0.4.5-0.20190127090136-055fb4bc9a6a

replace github.com/99designs/gqlgen/graphql => ../graphql

require (
	github.com/stretchr/testify v1.3.0
	// Actual dependencies
	github.com/vektah/gqlparser v1.1.0
)

module github.com/99designs/gqlgen/handler

require (
	github.com/99designs/gqlgen/complexity v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/graphql v0.4.5-0.20190203203210-e4679b668de0
)

replace (
	github.com/99designs/gqlgen => ../
	github.com/99designs/gqlgen/complexity => ../complexity
	github.com/99designs/gqlgen/graphql => ../graphql
)

require (
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/golang-lru v0.5.0
	github.com/stretchr/testify v1.3.0
	github.com/vektah/gqlparser v1.1.0
)

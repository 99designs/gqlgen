module github.com/99designs/gqlgen/handler

require (
	github.com/99designs/gqlgen/complexity v0.4.5-0.20190127090136-055fb4bc9a6a
	github.com/99designs/gqlgen/graphql v0.4.5-0.20190127090136-055fb4bc9a6a
)

replace (
	github.com/99designs/gqlgen => ../
	github.com/99designs/gqlgen/complexity => ../complexity
	github.com/99designs/gqlgen/graphql => ../graphql
)

require (
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/golang-lru v0.5.0
	github.com/vektah/gqlparser v1.1.0
)

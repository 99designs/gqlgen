---
title: "Using Gin to setup HTTP handlers"
description: Setting up HTTP handlers using Gin, a HTTP web framework written in Go.
linkTitle: Gin
menu: { main: { parent: 'recipes' } }
---

Gin is an excellent alternative for the `net/http` router. From their official [GitHub page](https://github.com/gin-gonic/gin):

> Gin is a web framework written in Go (Golang). It features a martini-like API with much better performance, up to 40 times faster thanks to httprouter. If you need performance and good productivity, you will love Gin.

Here are the steps to setup Gin and gqlgen together:

Install Gin:
```bash
$ go get gin
```

In your router file, define the handlers for the GraphQL and Playground endpoints in two different methods and tie then together in the Gin router:
```go
import (
    "github.com/99designs/gqlgen/handler"
    "github.com/gin-gonic/gin"
)

// Defining the Graphql handler
func graphqlHandler() gin.HandlerFunc {
    // NewExecutableSchema and Config are in the generated.go file
    // Resolver is in the resolver.go file
	h := handler.GraphQL(NewExecutableSchema(Config{Resolvers: &Resolver{}}))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// Defining the Playground handler
func playgroundHandler() gin.HandlerFunc {
	h := handler.Playground("GraphQL", "/query")

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {
    // Setting up Gin  
    r := gin.Default()
	r.POST("/query", graphqlHandler())
	r.GET("/", playgroundHandler())
    r.Run()
}  
```

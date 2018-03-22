---
linkTitle: Getting Started
title: Building graphql servers in golang
description: Get started building type-safe graphql servers in Golang using gqlgen  
menu: main
weight: -5
---

## Goal

The aim for this tutorial is to build a "todo" graphql server that can:

 - get a list of all todos
 - create new todos
 - mark off todos as they are completed

You can find the finished code for this tutorial [here](https://github.com/vektah/gqlgen-tutorials/tree/master/gettingstarted)

## Install gqlgen

Assuming you already have a working [go environment](https://golang.org/doc/install) you can simply go get:

```sh
go get github.com/vektah/gqlgen
```


## Define the schema

gqlgen is a schema-first library, so before touching any code we write out the API we want using the graphql 
[Schema Definition Language](http://graphql.org/learn/schema/). This usually goes into a file called schema.graphql  

```graphql
type Todo {
  id: ID!
  text: String!
  done: Boolean!
  user: User!
}

type User {
    id: ID!
    name: String!
}

type Query {
  todos: [Todo!]!
}

type Mutation {
  createTodo(text: String!): Todo!
}
```

## Generate the bindings

Now that we have defined the shape of our data, and what actions can be taken we can ask gqlgen to convert the schema into code:
```bash
mkdir graph
cd graph
gqlgen -schema ../schema.graphql
```

gqlgen should have created two new files `generated.go` and `models_gen.go`. If we take a peek in both we can see what the server has generated:

```go
// graph/generated.go
func MakeExecutableSchema(resolvers Resolvers) graphql.ExecutableSchema {
	return &executableSchema{resolvers}
}

type Resolvers interface {
	Mutation_createTodo(ctx context.Context, text string) (Todo, error)
	Query_todos(ctx context.Context) ([]Todo, error)
	Todo_user(ctx context.Context, it *Todo) (User, error)
}

// graph/models_gen.go
type Todo struct {
	ID     string
	Text   string
	Done   bool
	UserID string
}

type User struct {
	ID   string
	Name string
}

```

**Note**: ctx here is the golang context.Context, its used to pass per-request context like url params, tracing 
information, cancellation, and also the current selection set. This makes it more like the `info` argument in 
`graphql-js`. Because the caller will create an object to satisfy the interface, they can inject any dependencies in 
directly.

## Write the resolvers

Finally, we get to write some code! 

```go
// graph/graph.go
package graph

import (
	"context"
	"fmt"
	"math/rand"
)

type MyApp struct {
	todos []Todo
}

func (a *MyApp) Query_todos(ctx context.Context) ([]Todo, error) {
	return a.todos, nil
}

func (a *MyApp) Mutation_createTodo(ctx context.Context, text string) (Todo, error) {
	todo := Todo{
		Text:   text,
		ID:     fmt.Sprintf("T%d", rand.Int()),
		UserID: fmt.Sprintf("U%d", rand.Int()),
	}
	a.todos = append(a.todos, todo)
	return todo, nil
}

func (a *MyApp) Todo_user(ctx context.Context, it *Todo) (User, error) {
	return User{ID: it.UserID, Name: "user " + it.UserID}, nil
}
```

```go
// main.go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/vektah/gqlgen-tutorials/gettingstarted/graph"
	"github.com/vektah/gqlgen/handler"
)

func main() {
	app := &graph.MyApp{}
	http.Handle("/", handler.Playground("Todo", "/query"))
	http.Handle("/query", handler.GraphQL(graph.MakeExecutableSchema(app)))

	fmt.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```

We now have a working server, to start it:
```bash
go run *.go
```

then open http://localhost:8080 in a browser. here are some queries to try:
```graphql
mutation createTodo {
  createTodo(text:"test") {
    user {
      id
    }
    text
    done
  }
}

query findTodos {
  todos {
    text
    done
    user {
      name
    }
  }
}
```

## Customizing the models

Generated models are nice to get moving quickly, but you probably want control over them at some point. To do that
create a `graph/types.json`:
```json
{
  "User": "github.com/vektah/gqlgen-tutorials/gettingstarted/graph.User"
}
```

and create the model yourself:
```go
// graph/graph.go
type User struct {
	ID   string
	Name string
}
```

then regenerate, this time specifying the type map:

```bash
gqlgen -typemap types.json -schema ../schema.graphql
```

gqlgen will look at the user defined types and match the fields up finding fields and functions by matching names.


## Finishing touches

gqlgen is still unstable, and the APIs may change at any time. To prevent changes from ruining your day make sure
to lock your dependencies:

*Note*: If you dont have dep installed yet, you can get it [here](https://github.com/golang/dep)

```bash
dep init
dep ensure
go get github.com/vektah/gorunpkg
```

at the top of our main.go:
```go
//go:generate gorunpkg github.com/vektah/gqlgen -typemap types.json -out generated.go -package main

package main
```
**Note:** be careful formatting this, there must no space between the `//` and `go:generate`, and one empty line
between it and the `package main`.


This magic comment tells `go generate` what command to run when we want to regenerate our code. to do so run:
```go
go generate ./..
``` 

*gorunpkg* will build and run the version of gqlgen we just installed into vendor with dep. This makes sure
that everyone working on your project generates code the same way regardless which binaries are installed in their gopath.


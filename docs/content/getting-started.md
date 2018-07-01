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
go get -u github.com/vektah/gqlgen
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

input NewTodo {
  text: String!
  userId: String!
}

type Mutation {
  createTodo(input: NewTodo!): Todo!
}
```

Now lets write out our models:
graph/graph.go
`graph/graph.go`
```go
package graph


type User struct {
	ID   string
	Name string
}

type Todo struct {
	ID     string
	Text   string
	Done   bool
	UserID string
}

```

So we have our schema and our models. now we need to link them up:

`graph/types.json`
```json
{
  "User": "github.com/vektah/gqlgen-tutorials/gettingstarted/graph.User",
  "Todo": "github.com/vektah/gqlgen-tutorials/gettingstarted/graph.Todo"
}
``` 
This simply says, `User` in schema is backed by `graph.User` in go.


gqlgen then follows some pretty simple rules to match up all the fields:

 1. If there is a property with that name and type, use it
 2. If there is a method with that name and type, use it
 3. Otherwise, add it to the Resolvers interface. This is the magic.

## Generate the bindings


Lets generate the server now: 

```bash
mkdir graph
cd graph
gqlgen -typemap types.json -schema ../schema.graphql
```

gqlgen should have created two new files `generated.go` and `models_gen.go`. If we take a peek in both we can see what the server has generated:

```go
// graph/generated.go
func MakeExecutableSchema(resolvers Resolvers) graphql.ExecutableSchema {
	return &executableSchema{resolvers}
}

type Resolvers interface {
	Mutation_createTodo(ctx context.Context, input NewTodo) (Todo, error)
	Query_todos(ctx context.Context) ([]Todo, error)

	Todo_user(ctx context.Context, obj *Todo) (User, error) 
}

// graph/models_gen.go
type NewTodo struct {
	Text string `json:"text"`
	User int    `json:"user"`
}
```

Notice the `Todo_user` resolver? Thats gqlgen saying "I dont know how to get a User from a Todo, you tell me.". Its worked out everything else, for us.

For any missing models (like NewTodo) gqlgen will generate a go struct. This is usually only used for input types and one-off return values. Most of the time your types will be coming from the database, or an API client so binding is better than generating. 

## Write the resolvers

All thats left for us to do now is fill in the blanks in that interface:

`graph/graph.go`
```go
package graph

import (
	"context"
	"fmt"
	"math/rand"
)

type User struct {
	ID   string
	Name string
}

type Todo struct {
	ID     string
	Text   string
	Done   bool
	UserID string
}

type MyApp struct {
	todos []Todo
}

func (a *MyApp) Query_todos(ctx context.Context) ([]Todo, error) {
	return a.todos, nil
}

func (a *MyApp) Mutation_createTodo(ctx context.Context, input NewTodo) (Todo, error) {
	todo := Todo{
		Text:   input.Text,
		ID:     fmt.Sprintf("T%d", rand.Int()),
		UserID: input.UserId,
	}
	a.todos = append(a.todos, todo)
	return todo, nil
}

func (a *MyApp) Todo_user(ctx context.Context, it *Todo) (User, error) {
	return User{ID: it.UserID, Name: "user " + it.UserID}, nil
}
```

`main.go`
```go
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
go run main.go
```

then open http://localhost:8080 in a browser. here are some queries to try:
```graphql
mutation createTodo {
  createTodo(input:{text:"todo", userId:"1"}) {
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

## Finishing touches

gqlgen is still unstable, and the APIs may change at any time. To prevent changes from ruining your day make sure
to lock your dependencies:

*Note*: If you dont have dep installed yet, you can get it [here](https://github.com/golang/dep)

First uninstall the global version we grabbed earlier. This is a good way to prevent version mismatch footguns.

```bash
rm ~/go/bin/gqlgen
rm -rf ~/go/src/github.com/vektah/gqlgen
``` 

Next install gorunpkg, its kind of like npx but only searches vendor.

```bash
dep init
dep ensure
go get github.com/vektah/gorunpkg
```

Now at the top of our graph.go:
```go
//go:generate gorunpkg gqlgen -typemap types.json -schema ../schema.graphql

package graph
```
**Note:** be careful formatting this, there must no space between the `//` and `go:generate`, and one empty line
between it and the `package main`.


This magic comment tells `go generate` what command to run when we want to regenerate our code. to do so run:
```go
go generate ./...
``` 

*gorunpkg* will build and run the version of gqlgen we just installed into vendor with dep. This makes sure
that everyone working on your project generates code the same way regardless which binaries are installed in their gopath.


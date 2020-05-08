---
linkTitle: Getting Started
title: Building GraphQL servers in golang
description: Get started building type-safe GraphQL servers in Golang using gqlgen
menu: main
weight: -7
---

This tutorial will take you through the process of building a GraphQL server with gqlgen that can:

 - Return a list of todos
 - Create new todos
 - Mark off todos as they are completed

You can find the finished code for this tutorial [here](https://github.com/vektah/gqlgen-tutorials/tree/master/gettingstarted)

## Setup Project

Create a directory for your project, and initialise it as a Go Module:

```sh
$ mkdir gqlgen-todos
$ cd gqlgen-todos
$ go mod init github.com/[username]/gqlgen-todos
$ go get github.com/99designs/gqlgen
```

## Building the server

### Create the project skeleton

```bash
$ go run github.com/99designs/gqlgen init
```

This will create our suggested package layout. You can modify these paths in gqlgen.yml if you need to.
```
├── go.mod
├── go.sum
├── gqlgen.yml               - The gqlgen config file, knobs for controlling the generated code.
├── graph
│   ├── generated            - A package that only contains the generated runtime
│   │   └── generated.go
│   ├── model                - A package for all your graph models, generated or otherwise
│   │   └── models_gen.go
│   ├── resolver.go          - The root graph resolver type. This file wont get regenerated
│   ├── schema.graphqls      - Some schema. You can split the schema into as many graphql files as you like
│   └── schema.resolvers.go  - the resolver implementation for schema.graphql
└── server.go                - The entry point to your app. Customize it however you see fit
```

### Define your schema

gqlgen is a schema-first library — before writing code, you describe your API using the GraphQL
[Schema Definition Language](http://graphql.org/learn/schema/). By default this goes into a file called
`schema.graphql` but you can break it up into as many different files as you want.

The schema that was generated for us was:
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

### Implement the resolvers

`gqlgen generate` compares the schema file (`graph/schema.graphqls`) with the models `graph/model/*` and wherever it
can it will bind directly to the model.

If we take a look in `graph/schema.resolvers.go` we will see all the times that gqlgen couldn't match them up. For us
it was twice:

```go
func (r *mutationResolver) CreateTodo(ctx context.Context, input model.NewTodo) (*model.Todo, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Todos(ctx context.Context) ([]*model.Todo, error) {
	panic(fmt.Errorf("not implemented"))
}
```

We just need to implement these two methods to get our server working:

First we need somewhere to track our state, lets put it in `graph/resolver.go`:
```go
type Resolver struct{
	todos []*model.Todo
}
```
This is where we declare any dependencies for our app like our database, it gets initialized once in `server.go` when
we create the graph.

```go
func (r *mutationResolver) CreateTodo(ctx context.Context, input model.NewTodo) (*model.Todo, error) {
	todo := &model.Todo{
		Text:   input.Text,
		ID:     fmt.Sprintf("T%d", rand.Int()),
		User: &model.User{ID: input.UserID, Name: "user " + input.UserID},
	}
	r.todos = append(r.todos, todo)
	return todo, nil
}

func (r *queryResolver) Todos(ctx context.Context) ([]*model.Todo, error) {
	return r.todos, nil
}
```

We now have a working server, to start it:
```bash
go run server.go
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

### Dont eagerly fetch the user

This example is great, but in the real world fetching most objects is expensive. We dont want to load the User on the
todo unless the user actually asked for it. So lets replace the generated `Todo` model with something slightly more
realistic.

Create a new file called `graph/model/todo.go`
```go
package model

type Todo struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Done   bool   `json:"done"`
	UserID string `json:"user"`
}
```

> Note
>
> By default gqlgen will use any models in the model directory that match on name, this can be configured in `gqlgen.yml`.

And run `go run github.com/99designs/gqlgen generate`.

Now if we look in `graph/schema.resolvers.go` we can see a new resolver, lets implement it and fix `CreateTodo`.
```go
func (r *mutationResolver) CreateTodo(ctx context.Context, input model.NewTodo) (*model.Todo, error) {
	todo := &model.Todo{
		Text:   input.Text,
		ID:     fmt.Sprintf("T%d", rand.Int()),
		UserID: input.UserID, // fix this line
	}
	r.todos = append(r.todos, todo)
	return todo, nil
}

func (r *todoResolver) User(ctx context.Context, obj *model.Todo) (*model.User, error) {
	return &model.User{ID: obj.UserID, Name: "user " + obj.UserID}, nil
}
```

## Finishing touches

At the top of our `resolver.go`, between `package` and `import`, add the following line:

```go
//go:generate go run github.com/99designs/gqlgen
```

This magic comment tells `go generate` what command to run when we want to regenerate our code.  To run go generate recursively over your entire project, use this command:

```go
go generate ./...
```

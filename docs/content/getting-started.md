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

> Note
>
> This tutorial uses Go Modules and requires Go 1.11+.  If you want to use this tutorial without Go Modules, take a look at our [Getting Started Using dep]({{< ref "getting-started-dep.md" >}}) guide instead.

## Setup Project

Create a directory for your project, and initialise it as a Go Module:

```sh
$ mkdir gqlgen-todos
$ cd gqlgen-todos
$ go mod init github.com/[username]/gqlgen-todos
```

## Building the server

### Define the schema

gqlgen is a schema-first library — before writing code, you describe your API using the GraphQL 
[Schema Definition Language](http://graphql.org/learn/schema/). This usually goes into a file called `schema.graphql`:

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

### Create the project skeleton

```bash
$ go run github.com/99designs/gqlgen init
```

This has created an empty skeleton with all files you need:

 - `gqlgen.yml` — The gqlgen config file, knobs for controlling the generated code.
 - `generated.go` — The GraphQL execution runtime, the bulk of the generated code.
 - `models_gen.go` — Generated models required to build the graph. Often you will override these with your own models. Still very useful for input types.
 - `resolver.go` — This is where your application code lives. `generated.go` will call into this to get the data the user has requested. 
 - `server/server.go` — This is a minimal entry point that sets up an `http.Handler` to the generated GraphQL server.
 
### Create the database models

The generated model for Todo isn't right, it has a user embeded in it but we only want to fetch it if the user actually requested it. So instead lets make a new model in `todo.go`:

```go
package gqlgen_todos

type Todo struct {
	ID     string
	Text   string
	Done   bool
	UserID string
}
```

Next tell gqlgen to use this new struct by adding it to `gqlgen.yml`:

```yaml
models:
  Todo:
    model: github.com/[username]/gqlgen-todos.Todo
```

Regenerate by running:

```bash
$ go run github.com/99designs/gqlgen
```

> Note
>
> The verbose flag `-v` is here to show what gqlgen is doing. It has looked at all the fields on the model and found matching methods for all of them, except user. For user it has added a resolver to the interface you need to implement. *This is the magic that makes gqlgen work so well!*

### Implement the resolvers

The generated runtime has defined an interface for all the missing resolvers that we need to provide. Lets take a look in `generated.go`:

```go
func NewExecutableSchema(cfg Config) graphql.ExecutableSchema {}
	// ...
}

type Config struct {
	Resolvers  ResolverRoot
	// ...
}

type ResolverRoot interface {
	Mutation() MutationResolver
	Query() QueryResolver
	Todo() TodoResolver
}

type MutationResolver interface {
	CreateTodo(ctx context.Context, input NewTodo) (*Todo, error)
}
type QueryResolver interface {
	Todos(ctx context.Context) ([]Todo, error)
}
type TodoResolver interface {
	User(ctx context.Context, obj *Todo) (*User, error)
}
```

Notice the `TodoResolver.User` method? Thats gqlgen saying "I dont know how to get a User from a Todo, you tell me.".
Its worked out how to build everything else for us.

For any missing models (like `NewTodo`) gqlgen will generate a go struct. This is usually only used for input types and 
one-off return values. Most of the time your types will be coming from the database, or an API client so binding is
better than generating.

### Write the resolvers

This is a work in progress, we have a way to generate resolver stubs, but it cannot currently update existing code. We can force it to run again by deleting `resolver.go` and re-running gqlgen:

```bash
$ rm resolver.go
$ go run github.com/99designs/gqlgen
```

Now we just need to fill in the `not implemented` parts.  Update `resolver.go`

```go
package gqlgen_todos

import (
	context "context"
	"fmt"
	"math/rand"
)

type Resolver struct {
	todos []Todo
}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Todo() TodoResolver {
	return &todoResolver{r}
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreateTodo(ctx context.Context, input NewTodo) (*Todo, error) {
	todo := &Todo{
		Text:   input.Text,
		ID:     fmt.Sprintf("T%d", rand.Int()),
		UserID: input.UserID,
	}
	r.todos = append(r.todos, *todo)
	return todo, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Todos(ctx context.Context) ([]Todo, error) {
	return r.todos, nil
}

type todoResolver struct{ *Resolver }

func (r *todoResolver) User(ctx context.Context, obj *Todo) (*User, error) {
	return &User{ID: obj.UserID, Name: "user " + obj.UserID}, nil
}

```

We now have a working server, to start it:
```bash
go run server/server.go
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

At the top of our `resolver.go` add the following line:

```go
//go:generate go run github.com/99designs/gqlgen
```

This magic comment tells `go generate` what command to run when we want to regenerate our code.  To run go generate recursively over your entire project, use this command:

```go
go generate ./...
```

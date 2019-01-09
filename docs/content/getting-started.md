---
linkTitle: Getting Started
title: Building GraphQL servers in golang
description: Get started building type-safe GraphQL servers in Golang using gqlgen  
menu: main
weight: -5
---

This tutorial will take you through the process of building a GraphQL server with gqlgen that can:

 - Return a list of todos
 - Create new todos
 - Mark off todos as they are completed

You can find the finished code for this tutorial [here](https://github.com/vektah/gqlgen-tutorials/tree/master/gettingstarted)

## Install gqlgen

This article uses [`dep`](https://github.com/golang/dep) to install gqlgen.  [Follow the instructions for your environment](https://github.com/golang/dep) to install.

Assuming you already have a working [Go environment](https://golang.org/doc/install), create a directory for the project in your `$GOPATH`:

```sh
$ mkdir -p $GOPATH/src/github.com/[username]/gqlgen-todos
```

> Go Modules
>
> Currently `gqlgen` does not support Go Modules.  This is due to the [`loader`](https://godoc.org/golang.org/x/tools/go/loader) package, that also does not yet support Go Modules.  We are looking at solutions to this and the issue is tracked in Github.

Add the following file to your project under `scripts/gqlgen.go`:

```go
// +build ignore

package main

import "github.com/monzo/gqlgen/cmd"

func main() {
	cmd.Execute()
}
```

Lastly, initialise dep.  This will inspect any imports you have in your project, and pull down the latest tagged release.

```sh
$ dep init
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
$ go run scripts/gqlgen.go init
```

This has created an empty skeleton with all files you need:

 - `gqlgen.yml` — The gqlgen config file, knobs for controlling the generated code.
 - `generated.go` — The GraphQL execution runtime, the bulk of the generated code.
 - `models_gen.go` — Generated models required to build the graph. Often you will override these with your own models. Still very useful for input types.
 - `resolver.go` — This is where your application code lives. `generated.go` will call into this to get the data the user has requested. 
 - `server/server.go` — This is a minimal entry point that sets up an `http.Handler` to the generated GraphQL server.

 Now run dep ensure, so that we can ensure that the newly generated code's dependencies are all present:

 ```sh
 $ dep ensure
 ```
 
### Create the database models

The generated model for Todo isn't right, it has a user embeded in it but we only want to fetch it if the user actually requested it. So instead lets make a new model in `todo.go`:

```go
package gettingstarted

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
    model: github.com/[username]/gqlgen-todos/gettingstarted.Todo
```

Regenerate by running:

```bash
$ go run scripts/gqlgen.go -v
Unable to bind Todo.user to github.com/[username]/gqlgen-todos/gettingstarted.Todo
	no method named user
	no field named user
	Adding resolver method
```

> Note
>
> The verbose flag `-v` is here to show what gqlgen is doing. It has looked at all the fields on the model and found matching methods for all of them, except user. For user it has added a resolver to the interface you need to implement. *This is the magic that makes gqlgen work so well!*

### Implement the resolvers

The generated runtime has defined an interface for all the missing resolvers that we need to provide. Lets take a look in `generated.go`

```go
// NewExecutableSchema creates an ExecutableSchema from the ResolverRoot interface.
func NewExecutableSchema(cfg Config) graphql.ExecutableSchema {
	return &executableSchema{
		resolvers:  cfg.Resolvers,
		directives: cfg.Directives,
	}
}

type Config struct {
	Resolvers  ResolverRoot
	Directives DirectiveRoot
}

type ResolverRoot interface {
	Mutation() MutationResolver
	Query() QueryResolver
	Todo() TodoResolver
}

type DirectiveRoot struct {
}
type MutationResolver interface {
	CreateTodo(ctx context.Context, input NewTodo) (Todo, error)
}
type QueryResolver interface {
	Todos(ctx context.Context) ([]Todo, error)
}
type TodoResolver interface {
	User(ctx context.Context, obj *Todo) (User, error)
}
```

Notice the `TodoResolver.User` method? Thats gqlgen saying "I dont know how to get a User from a Todo, you tell me.".
Its worked out how to build everything else for us.

For any missing models (like NewTodo) gqlgen will generate a go struct. This is usually only used for input types and 
one-off return values. Most of the time your types will be coming from the database, or an API client so binding is
better than generating.

### Write the resolvers

This is a work in progress, we have a way to generate resolver stubs, but it cannot currently update existing code. We can force it to run again by deleting `resolver.go` and re-running gqlgen:

```bash
$ rm resolver.go
$ go run scripts/gqlgen.go
```

Now we just need to fill in the `not implemented` parts.  Update `resolver.go`

```go
//go:generate go run ./scripts/gqlgen.go

package gettingstarted

import (
	context "context"
	"fmt"
	"math/rand"
)

type Resolver struct{
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

func (r *mutationResolver) CreateTodo(ctx context.Context, input NewTodo) (Todo, error) {
	todo := Todo{
		Text:   input.Text,
		ID:     fmt.Sprintf("T%d", rand.Int()),
		UserID: input.UserID,
	}
	r.todos = append(r.todos, todo)
	return todo, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Todos(ctx context.Context) ([]Todo, error) {
	return r.todos, nil
}

type todoResolver struct{ *Resolver }

func (r *todoResolver) User(ctx context.Context, obj *Todo) (User, error) {
	return User{ID: obj.UserID, Name: "user " + obj.UserID}, nil
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
//go:generate go run scripts/gqlgen.go -v
```

This magic comment tells `go generate` what command to run when we want to regenerate our code.  To run go generate recursively over your entire project, use this command:

```go
go generate ./...
```

> Note
>
> Ensure that the path to your `gqlgen` binary is relative to the file the generate command is added to.

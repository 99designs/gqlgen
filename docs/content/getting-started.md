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
go get -u github.com/99designs/gqlgen github.com/vektah/gorunpkg
```

## Building the server

### Define the schema

gqlgen is a schema-first library, so before touching any code we write out the API we want using the graphql 
[Schema Definition Language](http://graphql.org/learn/schema/). This usually goes into a file called schema.graphql  

`schema.graphql`
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
$ gqlgen init
Exec "go run ./server/server.go" to start GraphQL server
```

This has created an empty skeleton with all files we need:

 - gqlgen.yml - The gqlgen config file, knobs for controlling the generated code.
 - generated.go - The graphql execution runtime, the bulk of the generated code
 - models_gen.go - Generated models required to build the graph. Often you will override these with models you write yourself. Still very useful for input types.
 - resolver.go - This is where your application code lives. generated.go will call into this to get the data the user has requested. 
 
### Create the database models

The generated model for Todo isnt quite right, it has a user embeded in it but we only want to fetch it if the user actually requested it. So lets make our own.

`todo.go`
```go
package gettingstarted

type Todo struct {
	ID     string
	Text   string
	Done   bool
	UserID string
}
```

And then tell gqlgen to use this new struct by adding this to the gqlgen.yml:
```yaml
models:
  Todo:
    model: github.com/vektah/gqlgen-tutorials/gettingstarted.Todo
```

and regenerate by running
```bash
$ gqlgen -v
Unable to bind Todo.user to github.com/vektah/gqlgen-tutorials/gettingstarted.Todo
	no method named user
	no field named user
	Adding resolver method
```
*note* we've used the verbose flag here to show what gqlgen is doing. Its looked at all the fields on our model and found matching methods for all of them, except user. For user it added a resolver to the interface we need to implement. This is the magic that makes gqlgen work so well. 

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

This is a work in progress, we have a way to generate resolver stubs, but it only cant currently update existing code. We can force it to run again by deleting `resolvers.go` and re-running gqlgen:
```bash
rm resolvers.go
gqlgen
```

Now we just need to fill in the `not implemented` parts


`graph/graph.go`
```go
//go:generate gorunpkg github.com/99designs/gqlgen

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

gqlgen is still unstable, and the APIs may change at any time. To prevent changes from ruining your day make sure
to lock your dependencies:

*Note*: If you dont have dep installed yet, you can get it [here](https://github.com/golang/dep)

First uninstall the global version we grabbed earlier. This is a good way to prevent version mismatch footguns.

```bash
rm ~/go/bin/gqlgen
rm -rf ~/go/src/github.com/99designs/gqlgen
``` 

Next install gorunpkg, its kind of like npx but only searches vendor.

```bash
dep init
dep ensure
```

At the top of our resolvers.go a go generate command was added that looks like this:
```go
//go:generate gorunpkg github.com/99designs/gqlgen
```

This magic comment tells `go generate` what command to run when we want to regenerate our code. to do so run:
```go
go generate ./...
``` 

*gorunpkg* will build and run the version of gqlgen we just installed into vendor with dep. This makes sure that everyone working on your project generates code the same way regardless which binaries are installed in their gopath.


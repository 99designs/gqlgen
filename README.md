# gqlgen ![CircleCI](https://circleci.com/gh/vektah/gqlgen.svg?style=svg)

This is a library for quickly creating strictly typed graphql servers in golang.

### Getting started

#### install gqlgen
```bash
go get github.com/vektah/gqlgen
```


#### define a schema
schema.graphql
```graphql schema
schema {
	query: Query
	mutation: Mutation
}

type Query {
	todos: [Todo!]!
}

type Mutation {
	createTodo(text: String!): Todo!
}

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
```


#### generate the bindings


gqlgen can then take the schema and generate all the code needed to execute incoming graphql queries in a safe,
strictly typed manner:
```bash
gqlgen -out generated.go -package main
```

If you look at the top of `generated.go` it has created an interface and some temporary models:

```go
func MakeExecutableSchema(resolvers Resolvers) graphql.ExecutableSchema {
	return &executableSchema{resolvers}
}

type Resolvers interface {
	Mutation_createTodo(ctx context.Context, text string) (Todo, error)
	Query_todos(ctx context.Context) ([]Todo, error)
	Todo_user(ctx context.Context, it *Todo) (User, error)
}

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

type executableSchema struct {
	resolvers Resolvers
}

func (e *executableSchema) Schema() *schema.Schema {
	return parsedSchema
}
```

Notice that only the scalar types were added to the model? Todo.user doesnt exist on the struct, instead a resolver 
method has been added. Resolver methods have a simple naming convention of {Type}_{field}.

You're probably thinking why not just have a method on the user struct? Well, you can. But its assumed it will be a 
getter method and wont be hitting the database, so parallel execution is disabled and you dont have access to any 
database context. Plus, your models probably shouldn't be responsible for fetching more data. To define methods on the
model you will need to copy it out of the generated code and define it in types.json.


**Note**: ctx here is the golang context.Context, its used to pass per-request context like url params, tracing 
information, cancellation, and also the current selection set. This makes it more like the `info` argument in 
`graphql-js`. Because the caller will create an object to satisfy the interface, they can inject any dependencies in 
directly.

#### write our resolvers
Now we need to join the edges of the graph up. 

main.go:
```go
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/vektah/gqlgen/handler"
)

type MyApp struct {
	todos []Todo
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

func (a *MyApp) Query_todos(ctx context.Context) ([]Todo, error) {
	return a.todos, nil
}

func (a *MyApp) Todo_user(ctx context.Context, it *Todo) (User, error) {
	return User{ID: it.UserID, Name: "user " + it.UserID}, nil
}

func main() {
	app := &MyApp{
		todos: []Todo{}, // this would normally be a reference to the db
	}
	http.Handle("/", handler.Playground("Dataloader", "/query"))
	http.Handle("/query", handler.GraphQL(MakeExecutableSchema(app)))

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

#### customizing the models or reusing your existing ones

Generated models are nice to get moving quickly, but you probably want control over them at some point. To do that
create a types.json, eg:
```json
{
  "Todo": "github.com/vektah/gettingstarted.Todo"
}
```

and create the model yourself:
```go
type Todo struct {
	ID     string
	Text   string
	done   bool
	userID string // I've made userID private now.
}

// lets define a getter too. it could also return an error if we needed. 
func (t Todo) Done() bool {
	return t.done
} 

```

then regenerate, this time specifying the type map:

```bash
gqlgen -out generated.go -package main -typemap types.json
```

gqlgen will look at the user defined types and match the fields up finding fields and functions by matching names.


#### Finishing touches

gqlgen is still unstable, and the APIs may change at any time. To prevent changes from ruining your day make sure
to lock your dependencies:

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


### Included custom scalar types

Included in gqlgen there are some custom scalar types that will just work out of the box.

- Time: An RFC3339 date as a quoted string
- Map: a json object represented as a map[string]interface{}. Useful change sets.

You are free to redefine these any way you want in types.json, see the [custom scalar example](./example/scalars).

### Prior art

#### neelance/graphql-go

The gold standard of graphql servers in golang. It provided the inspiration, and a good chunk of code for gqlgen. Its
strictly typed and uses your schema and some reflection to build up a resolver graph. The biggest downside is the amount
of work building up all of the resolvers, wrapping every object manually.

Reasons to use gqlgen instead:
 - We bind directly to your types, you dont need to bind manually https://github.com/neelance/graphql-go/issues/28
 - We show you the interface required, no guess work https://github.com/neelance/graphql-go/issues/159
 - We use separate resolvers for query and mutation https://github.com/neelance/graphql-go/issues/145
 - Code generation makes nil pointer juggling explicit, fixing issues like https://github.com/neelance/graphql-go/issues/125 
 - Code generating makes binding issues obvious https://github.com/neelance/graphql-go/issues/33
 - Separating the resolvers from the data graph means we only need gofuncs around database calls, reducing the cost of https://github.com/neelance/graphql-go/pull/102 
 - arrays work just fine https://github.com/neelance/graphql-go/issues/144
 - first class dataloader support, see examples/dataloader

https://github.com/neelance/graphql-go

#### graphql-go/graphql

With this library you write the schema using its internal DSL as go code, and bind in all your resolvers. No go type
information is used so you can dynamically define new schemas which could be useful for building schema stitching
servers at runtime.

Reasons to use gqlgen instead:
 - strict types. Why go to all the effort of defining gql schemas and then bind it to interface{} everywhere?
 - first class dataloader support, see examples/dataloader
 - horrible runtime error messages when you mess up defining your schema https://github.com/graphql-go/graphql/issues/234
 - reviewing schema changes written in a go dsl is really hard across teams

see https://github.com/graphql-go/graphql

#### Applifier/graphql-codegen and euforic/graphql-gen-go

Very similar idea, take your schema and generate the code from it.

gqlgen will build the entire execution environment statically, allowing go's type checker to validate everything across
the the graph. These two libraries generate resolvers that are loaded using reflection by the neelance library, so they
have most of the downsides of that with an added layer of complexity.

see https://github.com/Applifier/graphql-codegen and https://github.com/euforic/graphql-gen-go

# gqlgen [![CircleCI](https://circleci.com/gh/Vektah/gqlgen.svg?style=svg)](https://circleci.com/gh/Vektah/gqlgen)

This is a library for quickly creating strictly typed graphql servers in golang.

`dep ensure -add github.com/vektah/gqlgen`

Please use [dep](https://github.com/golang/dep) to pin your versions, the apis here should be considered unstable.

Ideally you should version the binary used to generate the code, as well as the library itself. Version mismatches
between the generated code and the runtime will be ugly. [gorunpkg](https://github.com/vektah/gorunpkg) makes this
as easy as:

Gopkg.toml
```toml
required = ["github.com/vektah/gqlgen"]  
```

then
```go
//go:generate gorunpkg github.com/vektah/gqlgen -out generated.go
```

#### Todo

 - [ ] opentracing
 - [ ] subscriptions

### Try it

Define your schema first:
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

Then define your models:
```go
package yourapp

type Todo struct {
	ID     string
	Text   string
	Done   bool
	UserID int
}

type User struct {
    ID string	
    Name string
}
```

Tell the generator how to map between the two in `types.json`
```json
{
  "Todo": "github.com/you/yourapp.Todo",
  "User": "github.com/you/yourapp.User"
}
```

Then generate the runtime from it:
```bash
gqlgen -out generated.go
```

At the top of the generated file will be an interface with the resolvers that are required to complete the graph:
```go
package yourapp

type Resolvers interface {
	Mutation_createTodo(ctx context.Context, text string) (Todo, error)

	Query_todos(ctx context.Context) ([]Todo, error)

	Todo_user(ctx context.Context, it *Todo) (User, error)
}
```

implement this interface, then create a server with by passing it into the generated code:
```go 
func main() {
	http.Handle("/query", graphql.Handler(gen.NewResolver(yourResolvers{})))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

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

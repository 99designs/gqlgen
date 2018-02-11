# gqlgen [![CircleCI](https://circleci.com/gh/Vektah/gqlgen.svg?style=svg)](https://circleci.com/gh/Vektah/gqlgen)

This is a library for quickly creating a strictly typed graphql servers in golang.

`go get -u github.com/vektah/gqlgen`
 
#### Try it

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

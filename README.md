# Grand Rounds Fork of GQLgen

This fork was created by:

1. Forking the repository
2. Resetting master of the fork to the tagged commit of the last stable release
   of gqlgen. At the time of writing this, it's tag v0.13.0 and commit 07c1f93.
3. Pushing -f to reset master of this fork the above commit.
3. Opening a pull request of the patch we've applied, to this fork, and merging
   the patch into the master branch of this fork.s

Note that means the master branch of this fork diverges from the official
repository. To update this fork to the latest version of gqlgen:

1. Merge the latest version tagged commit of upstream into master of this fork.
   **Don't** merge in the master branch of the main repository, just the commit
   of the next tagged version.

# gqlgen [![Continuous Integration](https://github.com/99designs/gqlgen/workflows/Continuous%20Integration/badge.svg)](https://github.com/99designs/gqlgen/actions) [![Read the Docs](https://badgen.net/badge/docs/available/green)](http://gqlgen.com/) [![GoDoc](https://godoc.org/github.com/99designs/gqlgen?status.svg)](https://godoc.org/github.com/99designs/gqlgen)

![gqlgen](https://user-images.githubusercontent.com/46195831/89802919-0bb8ef00-db2a-11ea-8ba4-88e7a58b2fd2.png)

## What is gqlgen?

[gqlgen](https://github.com/99designs/gqlgen) is a Go library for building GraphQL servers without any fuss.<br/> 

- **gqlgen is based on a Schema first approach** — You get to Define your API using the GraphQL [Schema Definition Language](http://graphql.org/learn/schema/).
- **gqlgen prioritizes Type safety** — You should never see `map[string]interface{}` here.
- **gqlgen enables Codegen** — We generate the boring bits, so you can focus on building your app quickly.

Still not convinced enough to use **gqlgen**? Compare **gqlgen** with other Go graphql [implementations](https://gqlgen.com/feature-comparison/)

## Getting Started
- To install gqlgen run the command `go get github.com/99designs/gqlgen` in your project directory.<br/> 
- You could initialize a new project using the recommended folder structure by running this command `go run github.com/99designs/gqlgen init`.

You could find a more comprehensive guide to help you get started [here](https://gqlgen.com/getting-started/).<br/>
We also have a couple of real-world [examples](https://github.com/99designs/gqlgen/tree/master/example) that show how to GraphQL applications with **gqlgen** seamlessly,
You can see these [examples](https://github.com/99designs/gqlgen/tree/master/example) here or visit [godoc](https://godoc.org/github.com/99designs/gqlgen).

## Reporting Issues

If you think you've found a bug, or something isn't behaving the way you think it should, please raise an [issue](https://github.com/99designs/gqlgen/issues) on GitHub.

## Contributing

We welcome contributions, Read our [Contribution Guidelines](https://github.com/99designs/gqlgen/blob/master/CONTRIBUTING.md) to learn more about contributing to **gqlgen**
## Frequently asked questions

### How do I prevent fetching child objects that might not be used?

When you have nested or recursive schema like this:

```graphql
type User {
  id: ID!
  name: String!
  friends: [User!]!
}
```

You need to tell gqlgen that it should only fetch friends if the user requested it. There are two ways to do this;

- #### Using Custom Models

Write a custom model that omits the friends field:

```go
type User struct {
  ID int
  Name string
}
```

And reference the model in `gqlgen.yml`:

```yaml
# gqlgen.yml
models:
  User:
    model: github.com/you/pkg/model.User # go import path to the User struct above
```

- #### Using Explicit Resolvers

If you want to Keep using the generated model, mark the field as requiring a resolver explicitly in `gqlgen.yml` like this:

```yaml
# gqlgen.yml
models:
  User:
    fields:
      friends:
        resolver: true # force a resolver to be generated
```

After doing either of the above and running generate we will need to provide a resolver for friends:

```go
func (r *userResolver) Friends(ctx context.Context, obj *User) ([]*User, error) {
  // select * from user where friendid = obj.ID
  return friends,  nil
}
```

You can also use inline config with directives to achieve the same result

```graphql
directive @goModel(model: String, models: [String!]) on OBJECT
    | INPUT_OBJECT
    | SCALAR
    | ENUM
    | INTERFACE
    | UNION

directive @goField(forceResolver: Boolean, name: String) on INPUT_FIELD_DEFINITION
    | FIELD_DEFINITION

type User @goModel(model: "github.com/you/pkg/model.User") {
    id: ID!         @goField(name: "todoId")
    friends: [User!]!   @goField(forceResolver: true)
}
```

### Can I change the type of the ID from type String to Type Int?

Yes! You can by remapping it in config as seen below:

```yaml
models:
  ID: # The GraphQL type ID is backed by
    model:
      - github.com/99designs/gqlgen/graphql.IntID # An go integer
      - github.com/99designs/gqlgen/graphql.ID # or a go string
```

This means gqlgen will be able to automatically bind to strings or ints for models you have written yourself, but the
first model in this list is used as the default type and it will always be used when:

- Generating models based on schema
- As arguments in resolvers

There isn't any way around this, gqlgen has no way to know what you want in a given context.

## Other Resources

- [Christopher Biscardi @ Gophercon UK 2018](https://youtu.be/FdURVezcdcw)
- [Introducing gqlgen: a GraphQL Server Generator for Go](https://99designs.com.au/blog/engineering/gqlgen-a-graphql-server-generator-for-go/)
- [Dive into GraphQL by Iván Corrales Solera](https://medium.com/@ivan.corrales.solera/dive-into-graphql-9bfedf22e1a)
- [Sample Project built on gqlgen with Postgres by Oleg Shalygin](https://github.com/oshalygin/gqlgen-pg-todo-example)

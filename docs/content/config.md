---
linkTitle: Configuration
title: How to configure gqlgen using gqlgen.yml
description: How to configure gqlgen using gqlgen.yml
menu: main
weight: -5
---

gqlgen can be configured using a `gqlgen.yml` file, by default it will be loaded from the current directory, or any parent directory.

Example:
```yml
# You can pass a single schema file
schema: schema.graphql

# Or multiple files
schema:
 - schema.graphql
 - user.graphql

# Or you can use globs
schema:
 - "*.graphql"

# Or globs from a root directory
schema:
 - "schema/**/*.graphql"

# Let gqlgen know where to put the generated server
exec:
  filename: graph/generated/generated.go
  package: generated

# Let gqlgen know where to put the generated models (if any)
model:
  filename: models/generated.go
  package: models

# Optional, turns on resolver stub generation
resolver:
  filename: resolver.go # where to write them
  type: Resolver  # what's the resolver root implementation type called?

# Optional, turns on binding to field names by tag provided
struct_tag: json

# Optional, set to true if you prefer []*Thing over []Thing
omit_slice_element_pointers: false

# Instead of listing out every model like below, you can automatically bind to any matching types
# within the given path by using `model: User` or `model: models.User`. EXPERIMENTAL in v0.9.1
autobind:
 - github.com/my/app/models

# Tell gqlgen about any existing models you want to reuse for
# graphql. These normally come from the db or a remote api.
models:
  User:
    model: models.User # can use short paths when the package is listed in autobind
  Todo:
    model: github.com/my/app/models.Todo # or full paths if you need to go elsewhere
    fields:
      id:
        resolver: true # force a resolver to be generated
        fieldName: todoId # bind to a different go field name
  # model also accepts multiple backing go types. When mapping onto structs
  # any of these types can be used, the first one is used as the default for
  # resolver args.
  ID:
    model:
      - github.com/99designs/gqlgen/graphql.IntID
      - github.com/99designs/gqlgen/graphql.ID
```

Everything has defaults, so add things as you need.

## Inline config with directives

gqlgen ships with some builtin directives that make it a little easier to manage wiring.

To start using them you first need to define them:
```graphql
directive @goModel(model: String, models: [String!]) on OBJECT 
    | INPUT_OBJECT 
    | SCALAR 
    | ENUM 
    | INTERFACE 
    | UNION

directive @goField(forceResolver: Boolean, name: String) on INPUT_FIELD_DEFINITION 
    | FIELD_DEFINITION
```
  
> Here be dragons
>
> gqlgen doesnt currently support user-configurable directives for SCALAR, ENUM, INTERFACE or UNION. This only works
> for internal directives. You can track the progress [here](https://github.com/99designs/gqlgen/issues/760)

Now you can use these directives when defining types in your schema:

```graphql
type User @goModel(model:"github.com/my/app/models.User") {
  id:   ID!	    @goField(name:"todoId")
  name: String! @goField(forceResolver: true)
}
```

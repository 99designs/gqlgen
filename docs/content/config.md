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
# Where are all the schema files located? globs are supported eg  src/**/*.graphqls
schema:
  - graph/*.graphqls

# Where should the generated server code go?
exec:
  filename: graph/generated/generated.go
  package: generated

# Enable Apollo federation support
federation:
  filename: graph/generated/federation.go
  package: generated

# Where should any generated models go?
model:
  filename: graph/model/models_gen.go
  package: model

# Where should the resolver implementations go?
resolver:
  layout: follow-schema
  dir: graph
  package: graph
  filename_template: "{name}.resolvers.go"

# Optional: turn on use ` + "`" + `gqlgen:"fieldName"` + "`" + ` tags in your models
# struct_tag: json

# Optional: turn on to use []Thing instead of []*Thing
# omit_slice_element_pointers: false

# Optional: set to speed up generation time by not performing a final validation pass.
# skip_validation: true

# gqlgen will search for any type names in the schema in these go packages
# if they match it will use them, otherwise it will generate them.
autobind:
  - "github.com/your/app/graph/model"

# This section declares type mapping between the GraphQL and go type systems
#
# The first line in each type will be used as defaults for resolver arguments and
# modelgen, the others will be allowed when binding to fields. Configure them to
# your liking
models:
  ID:
    model:
      - github.com/99designs/gqlgen/graphql.ID
      - github.com/99designs/gqlgen/graphql.Int
      - github.com/99designs/gqlgen/graphql.Int64
      - github.com/99designs/gqlgen/graphql.Int32
  Int:
    model:
      - github.com/99designs/gqlgen/graphql.Int
      - github.com/99designs/gqlgen/graphql.Int64
      - github.com/99designs/gqlgen/graphql.Int32

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
type User @goModel(model: "github.com/my/app/models.User") {
    id: ID!         @goField(name: "todoId")
    name: String!   @goField(forceResolver: true)
}
```

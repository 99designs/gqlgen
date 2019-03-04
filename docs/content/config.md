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

# Tell gqlgen about any existing models you want to reuse for
# graphql. These normally come from the db or a remote api.
models:
  User:
    model: github.com/my/app/models.User
  Todo:
    model: github.com/my/app/models.Todo
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


---
linkTitle: Configuration
title: How to configure gqlgen using gqlgen.yml
description: How to configure gqlgen using gqlgen.yml
menu: main
weight: -7
---

gqlgen can be configured using a `gqlgen.yml` file, by default it will be loaded from the current directory, or any parent directory.

Example:
```yml
schema: schema.graphql

# Let gqlgen know where to put the generated server
exec:
  filename: graph/generated/generated.go
  package: generated

# Let gqlgen know where to the generated models (if any)
model:
  filename: models/generated.go
  package: models

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
```

Everything has defaults, so add things as you need.


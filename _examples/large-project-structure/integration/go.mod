module github.com/99designs/gqlgen/_examples/large-project-structure/integration

go 1.24.1

require github.com/99designs/gqlgen/_examples/large-project-structure/main v0.0.0

replace github.com/99designs/gqlgen/_examples/large-project-structure/main => ../main

replace github.com/99designs/gqlgen/_examples/large-project-structure/shared => ../shared

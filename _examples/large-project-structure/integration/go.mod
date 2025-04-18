module github.com/99designs/gqlgen/_examples/large-project-structure/integration

go 1.24.1

require github.com/99designs/gqlgen/_examples/large-project-structure/main v0.0.0

replace github.com/99designs/gqlgen/_examples/large-project-structure/main => ../main

replace github.com/99designs/gqlgen/_examples/large-project-structure/shared => ../shared

require (
	github.com/99designs/gqlgen v0.17.70 // indirect
	github.com/agnivade/levenshtein v1.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/sosodev/duration v1.3.1 // indirect
	github.com/vektah/gqlparser/v2 v2.5.23 // indirect
)

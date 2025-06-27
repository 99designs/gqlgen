module github.com/99designs/gqlgen/_examples/large-project-structure/main

go 1.24.1

require (
	github.com/99designs/gqlgen v0.17.70
	github.com/99designs/gqlgen/_examples/large-project-structure/integration v0.0.0-00010101000000-000000000000
	github.com/99designs/gqlgen/_examples/large-project-structure/shared v0.0.0
	github.com/vektah/gqlparser/v2 v2.5.23
)

replace github.com/99designs/gqlgen/_examples/large-project-structure/shared => ../shared

replace github.com/99designs/gqlgen/_examples/large-project-structure/integration => ../integration

require (
	github.com/agnivade/levenshtein v1.2.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.5 // indirect
	github.com/go-viper/mapstructure/v2 v2.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sosodev/duration v1.3.1 // indirect
	github.com/urfave/cli/v2 v2.27.6 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

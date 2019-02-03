module github.com/99designs/gqlgen/cmd

require github.com/99designs/gqlgen v0.7.1

replace (
	github.com/99designs/gqlgen => ../
	github.com/99designs/gqlgen/codegen => ../codegen
	github.com/99designs/gqlgen/codegen/config => ../codegen/config
	github.com/99designs/gqlgen/codegen/templates => ../codegen/templates
	github.com/99designs/gqlgen/complexity => ../complexity
	github.com/99designs/gqlgen/graphql => ../graphql
	github.com/99designs/gqlgen/graphql/introspection => ../graphql/introspection
	github.com/99designs/gqlgen/handler => ../handler
	github.com/99designs/gqlgen/internal/code => ../internal/code
	github.com/99designs/gqlgen/internal/imports => ../internal/imports
	github.com/99designs/gqlgen/plugin => ../plugin
	github.com/99designs/gqlgen/plugin/modelgen => ../plugin/modelgen
)

require (
	github.com/99designs/gqlgen/codegen/config v0.4.5-0.20190127090136-055fb4bc9a6a
	github.com/99designs/gqlgen/graphql v0.4.5-0.20190127090136-055fb4bc9a6a
	github.com/99designs/gqlgen/handler v0.4.5-0.20190127090136-055fb4bc9a6a
	github.com/pkg/errors v0.8.1
	github.com/urfave/cli v1.20.0
	gopkg.in/yaml.v2 v2.2.2
)

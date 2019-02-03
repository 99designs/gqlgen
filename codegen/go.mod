module github.com/99designs/gqlgen/codegen

require (
	github.com/99designs/gqlgen/codegen/config v0.4.5-0.20190127090136-055fb4bc9a6a
	github.com/99designs/gqlgen/codegen/templates v0.4.5-0.20190127090136-055fb4bc9a6a
)

replace (
	github.com/99designs/gqlgen/codegen/config => ./config
	github.com/99designs/gqlgen/codegen/templates => ./templates
	github.com/99designs/gqlgen/internal/code => ../internal/code
	github.com/99designs/gqlgen/internal/imports => ../internal/imports
)

require (
	github.com/99designs/gqlgen/internal/code v0.4.5-0.20190127090136-055fb4bc9a6a
	github.com/pkg/errors v0.8.1
	github.com/vektah/gqlparser v1.1.0
)

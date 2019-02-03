module github.com/99designs/gqlgen/plugin

require github.com/99designs/gqlgen/codegen/config v0.4.5-0.20190127090136-055fb4bc9a6a

replace (
	github.com/99designs/gqlgen/codegen/config => ../codegen/config
	github.com/99designs/gqlgen/internal/code => ../internal/code
)

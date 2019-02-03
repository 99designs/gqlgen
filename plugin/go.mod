module github.com/99designs/gqlgen/plugin

require github.com/99designs/gqlgen/codegen/config v0.4.5-0.20190203203210-e4679b668de0

replace (
	github.com/99designs/gqlgen/codegen/config => ../codegen/config
	github.com/99designs/gqlgen/internal/code => ../internal/code
)

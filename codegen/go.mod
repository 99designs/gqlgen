module github.com/99designs/gqlgen/codegen

require (
	github.com/99designs/gqlgen/codegen/config v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/codegen/templates v0.4.5-0.20190203203210-e4679b668de0
)

replace (
	github.com/99designs/gqlgen/codegen/config => ./config
	github.com/99designs/gqlgen/codegen/templates => ./templates
	github.com/99designs/gqlgen/internal/code => ../internal/code
	github.com/99designs/gqlgen/internal/imports => ../internal/imports
)

require (
	github.com/99designs/gqlgen/internal/code v0.4.5-0.20190203203210-e4679b668de0
	github.com/pkg/errors v0.8.1
	github.com/vektah/gqlparser v1.1.0
)

module github.com/99designs/gqlgen/plugin/modelgen

// Rewrite

require (
	github.com/99designs/gqlgen/codegen/config v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/codegen/templates v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/plugin v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/graphql v0.4.5-0.20190203203210-e4679b668de0
)

replace (
	github.com/99designs/gqlgen/codegen/config => ../../codegen/config
	github.com/99designs/gqlgen/codegen/templates => ../../codegen/templates
	github.com/99designs/gqlgen/internal/code => ../../internal/code
	github.com/99designs/gqlgen/internal/imports => ../../internal/imports
	github.com/99designs/gqlgen/plugin => ../../plugin
	github.com/99designs/gqlgen/graphql => ../../graphql
)

require (
	github.com/stretchr/testify v1.3.0
	github.com/vektah/gqlparser v1.1.0
)

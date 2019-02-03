module github.com/99designs/gqlgen/codegen/config

require github.com/99designs/gqlgen/internal/code v0.4.5-0.20190203203210-e4679b668de0

replace (
	github.com/99designs/gqlgen/graphql => ../../graphql
	github.com/99designs/gqlgen/internal/code => ../../internal/code
)

require (
	github.com/99designs/gqlgen v0.7.1 // indirect
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	github.com/vektah/gqlparser v1.1.0
	golang.org/x/tools v0.0.0-20190202235157-7414d4c1f71c
	gopkg.in/yaml.v2 v2.2.2
)

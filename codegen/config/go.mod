module github.com/99designs/gqlgen/codegen/config

require github.com/99designs/gqlgen/internal/code v0.4.5-0.20190127090136-055fb4bc9a6a

replace github.com/99designs/gqlgen/internal/code => ../../internal/code

require (
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	github.com/vektah/gqlparser v1.1.0
	golang.org/x/tools v0.0.0-20190202235157-7414d4c1f71c
	gopkg.in/yaml.v2 v2.2.2
)

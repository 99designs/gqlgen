module github.com/99designs/gqlgen/internal/imports

require golang.org/x/tools v0.0.0-20190202235157-7414d4c1f71c

require (
	github.com/99designs/gqlgen/internal/code v0.4.5-0.20190127090136-055fb4bc9a6a
	github.com/stretchr/testify v1.3.0
)

replace github.com/99designs/gqlgen/internal/code => ../code

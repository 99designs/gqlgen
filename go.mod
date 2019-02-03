module github.com/99designs/gqlgen

// Rewrite
require (
	github.com/99designs/gqlgen/codegen v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/codegen/config v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/codegen/templates v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/graphql v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/graphql/introspection v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/handler v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/plugin v0.4.5-0.20190203203210-e4679b668de0
	github.com/99designs/gqlgen/plugin/modelgen v0.4.5-0.20190203203210-e4679b668de0
)

replace (
	github.com/99designs/gqlgen/codegen => ./codegen
	github.com/99designs/gqlgen/codegen/config => ./codegen/config
	github.com/99designs/gqlgen/codegen/templates => ./codegen/templates
	github.com/99designs/gqlgen/complexity => ./complexity
	github.com/99designs/gqlgen/graphql => ./graphql
	github.com/99designs/gqlgen/graphql/introspection => ./graphql/introspection
	github.com/99designs/gqlgen/handler => ./handler
	github.com/99designs/gqlgen/internal/code => ./internal/code
	github.com/99designs/gqlgen/internal/imports => ./internal/imports
	github.com/99designs/gqlgen/plugin => ./plugin
	github.com/99designs/gqlgen/plugin/modelgen => ./plugin/modelgen
)

// Actual dependencies
require (
	github.com/go-chi/chi v3.3.2+incompatible
	github.com/gogo/protobuf v1.2.0 // indirect
	github.com/gorilla/mux v1.7.0 // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/mitchellh/mapstructure v0.0.0-20180203102830-a4e142e9c047
	github.com/opentracing/basictracer-go v1.0.0 // indirect
	github.com/opentracing/opentracing-go v1.0.2
	github.com/pkg/errors v0.8.1
	github.com/rs/cors v1.6.0
	github.com/shurcooL/httpfs v0.0.0-20181222201310-74dc9339e414 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20181202132449-6a9ea43bcacd // indirect
	github.com/stretchr/testify v1.3.0
	github.com/vektah/dataloaden v0.2.0
	github.com/vektah/gqlparser v1.1.0
	golang.org/x/net v0.0.0-20190125091013-d26f9f9a57f3 // indirect
	golang.org/x/tools v0.0.0-20190202235157-7414d4c1f71c
	sourcegraph.com/sourcegraph/appdash v0.0.0-20180110180208-2cc67fd64755
	sourcegraph.com/sourcegraph/appdash-data v0.0.0-20151005221446-73f23eafcf67 // indirect
)

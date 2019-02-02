module github.com/99designs/gqlgen

// Rewrite
require (
	github.com/99designs/gqlgen/complexity v0.4.5-0.20190127090136-055fb4bc9a6a
	github.com/99designs/gqlgen/graphql v0.4.5-0.20190127090136-055fb4bc9a6a
	github.com/99designs/gqlgen/graphql/introspection v0.4.5-0.20190127090136-055fb4bc9a6a
	github.com/99designs/gqlgen/handler v0.4.5-0.20190127090136-055fb4bc9a6a
)

replace (
	github.com/99designs/gqlgen/complexity => ./complexity
	github.com/99designs/gqlgen/graphql => ./graphql
	github.com/99designs/gqlgen/graphql/introspection => ./graphql/introspection
	github.com/99designs/gqlgen/handler => ./handler
)

// Actual dependencies
require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-chi/chi v3.3.2+incompatible
	github.com/gogo/protobuf v1.0.0 // indirect
	github.com/gorilla/context v0.0.0-20160226214623-1ea25387ff6f // indirect
	github.com/gorilla/mux v1.6.1 // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/hashicorp/golang-lru v0.5.0
	github.com/kr/pretty v0.1.0 // indirect
	github.com/mitchellh/mapstructure v0.0.0-20180203102830-a4e142e9c047
	github.com/opentracing/basictracer-go v1.0.0 // indirect
	github.com/opentracing/opentracing-go v1.0.2
	github.com/pkg/errors v0.8.1
	github.com/rs/cors v1.6.0
	github.com/shurcooL/httpfs v0.0.0-20171119174359-809beceb2371 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20180121065927-ffb13db8def0 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/urfave/cli v1.20.0
	github.com/vektah/dataloaden v0.2.0
	github.com/vektah/gqlparser v1.1.0
	golang.org/x/net v0.0.0-20180404174746-b3c676e531a6 // indirect
	golang.org/x/tools v0.0.0-20190125232054-d66bd3c5d5a6
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v2 v2.2.2
	sourcegraph.com/sourcegraph/appdash v0.0.0-20180110180208-2cc67fd64755
	sourcegraph.com/sourcegraph/appdash-data v0.0.0-20151005221446-73f23eafcf67 // indirect
)

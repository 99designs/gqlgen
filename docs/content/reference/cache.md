---
title: "Caching"
description:
linkTitle: "Caching"
menu: { main: { parent: 'reference' } }
---

Gqlgen provides some cache capabilities that provide ways to set caching directives in `extensions.cacheControl`
and HTTP `Cache-Control` header.

See more in `/example/cachecontrol`.

## Usage


### Enable

To enable cache capabilities, you need to use `cache.Extension` like above:

```go
package main

import (
	"github.com/99designs/gqlgen/graphql/handler/cache"

	"github.com/99designs/gqlgen/example/cachecontrol/graph"
	"github.com/99designs/gqlgen/example/cachecontrol/graph/generated"
	"github.com/99designs/gqlgen/graphql/handler"
)

func main() {
	// Building your server
	cfg := generated.Config{Resolvers: &graph.Resolver{}}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(cfg))

	// Enable cache extensions
	srv.Use(cache.Extension{})

	//...
}
```

the Cache extension is a `graphql.ResponseInterceptor` that provides cache control mechanism in each request context
and inject `cacheControl` into `graphql.Response`.

### Set cache hints

After you enable `cache.Extension`, you can set cache hints using `cache.SetHint` function in your resolvers.

```go
import (
	"github.com/99designs/gqlgen/graphql/handler/cache"
	// ...
)

func (r *commentResolver) Post(ctx context.Context, obj *model.Comment) (*model.Post, error) {
    post, err := // getting post by comment
    if err != nil {
        return nil, err
	}

	// Set a CacheHint
	cache.SetHint(ctx, cache.ScopePublic, 10*time.Second)

	return post, nil
}
```

### CDN Caching

It's possible to enable the Gqlgen to provide a `Cache-Control` header based on your cache hints in `GET` or `POST` requests.
To do it you need to enable on `transport.GET` or `transport.POST`:

```go

func main() {

	// ... setup server

	srv.AddTransport(transport.GET{EnableCache: true})
	srv.AddTransport(transport.POST{EnableCache: true})

	// ... do more things

}
```

Doing it, Gqlgen write the lowest max-age defined in cacheControl extensions.

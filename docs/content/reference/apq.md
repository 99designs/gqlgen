---
title: "Automatic persisted queries"
description:   
linkTitle: "APQ"
menu: { main: { parent: 'reference' } }
---

When you work with GraphQL by default your queries are transferred with every request. That can waste significant
bandwidth. To avoid that you can use Automatic Persisted Queries (APQ).

With APQ you send only query hash to the server. If hash is not found on a server then client makes a second request
to register query hash with original query on a server.

## Usage

In order to enable Automatic Persisted Queries you need to change your client. For more information see 
[Automatic Persisted Queries Link](https://github.com/apollographql/apollo-link-persisted-queries) documentation.

For the server you need to implement `PersistedQueryCache` interface and pass instance to 
`handler.EnablePersistedQueryCache` option.

See example using [go-redis](https://github.com/go-redis/redis) package below:
```go
import (
	"context"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

type Cache struct {
	client redis.UniversalClient
	ttl    time.Duration
}

const apqPrefix = "apq:"

func NewCache(redisAddress string, password string, ttl time.Duration) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
	})

	err := client.Ping().Err()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &Cache{client: client, ttl: ttl}, nil
}

func (c *Cache) Add(ctx context.Context, hash string, query string) {
	c.client.Set(apqPrefix + hash, query, c.ttl)
}

func (c *Cache) Get(ctx context.Context, hash string) (string, bool) {
	s, err := c.client.Get(apqPrefix + hash).Result()
	if err != nil {
		return "", false
	}
	return s, true
}

func main() {
	cache, err := NewCache(cfg.RedisAddress, 24*time.Hour)
	if err != nil {
		log.Fatalf("cannot create APQ redis cache: %v", err)
	}
	
	c := Config{ Resolvers: &resolvers{} }
	gqlHandler := handler.GraphQL(
		blog.NewExecutableSchema(c),
		handler.EnablePersistedQueryCache(cache),
	)
	http.Handle("/query", gqlHandler)
}
```

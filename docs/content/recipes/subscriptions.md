---
title: "Subscriptions"
description: Subscriptions allow for streaming real-time events to your clients. This is how to do that with gqlgen.
linkTitle: "Subscriptions"
menu: { main: { parent: 'recipes' } }
---

GraphQL Subscriptions allow you to stream events to your clients in real-time.
This is easy to do in gqlgen and this recipe will show you how to setup a quick example.

## Preparation

This recipe starts with the empty project after the quick start steps were followed.
Although the steps are the same in an existing, more complex projects you will need
to be careful to configure routing correctly.

In this recipe you will learn how to

1. add WebSocket Transport to your server
2. add the `Subscription` type to your schema
3. implement a real-time resolver.

## Adding WebSocket Transport

To send real-time data to clients, your GraphQL server needs to have an open connection
with the client. This is done using WebSockets.

To add the WebSocket transport change your `main.go` by calling `AddTransport(&transport.Websocket{})`
on your query handler.

**If you are using an external router, remember to send *ALL* `/query`-requests to your handler!**
**Not just POST requests!**

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/example/test/graph"
	"github.com/example/test/graph/generated"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))

	srv.AddTransport(&transport.Websocket{}) // <---- This is the important part!

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## Adding Subscriptions to your Schema

Next you'll have to define the subscriptions in your schema in the `Subscription` top-level type.

```graphql
"""
Make sure you have at least something in your `Query` type.
If you don't have a query the playground will be unable
to introspect your schema!
"""
type Query {
  placeholder: String
}

"""
`Time` is a simple type only containing the current time as
a unix epoch timestamp and a string timestamp.
"""
type Time {
  unixTime: Int!
  timeStamp: String!
}

"""
`Subscription` is where all the subscriptions your clients can
request. You can use Schema Directives like normal to restrict
access.
"""
type Subscription {
  """
  `currentTime` will return a stream of `Time` objects.
  """
  currentTime: Time!
}
```

## Implementing your Resolver

After regenerating your code with `go run github.com/99designs/gqlgen generate` you'll find a
new resolver for your subscription. It will look like any other resolver, except it expects
a `<-chan *model.Time` (or whatever your type is). This is a
[channel](https://go.dev/tour/concurrency/2). Channels in Go are used to send objects to a
single receiver.

The resolver for our example `currentTime` subscription looks as follows:

```go
// CurrentTime is the resolver for the currentTime field.
func (r *subscriptionResolver) CurrentTime(ctx context.Context) (<-chan *model.Time, error) {
	// First you'll need to `make()` your channel. Use your type here!
	ch := make(chan *model.Time)

	// You can (and probably should) handle your channels in a central place outside of `schema.resolvers.go`.
	// For this example we'll simply use a Goroutine with a simple loop.
	go func() {
		// Handle deregistration of the channel here. Note the `defer`
    defer close(ch)

		for {
			// In our example we'll send the current time every second.
			time.Sleep(1 * time.Second)
			fmt.Println("Tick")

			// Prepare your object.
			currentTime := time.Now()
			t := &model.Time{
				UnixTime:  int(currentTime.Unix()),
				TimeStamp: currentTime.Format(time.RFC3339),
			}

			// The subscription may have got closed due to the client disconnecting.
			// Hence we do send in a select block with a check for context cancellation.
			// This avoids goroutine getting blocked forever or panicking,
			select {
			case <-ctx.Done(): // This runs when context gets cancelled. Subscription closes.
				fmt.Println("Subscription Closed")
				// Handle deregistration of the channel here. `close(ch)`
				return // Remember to return to end the routine.
			
			case ch <- t: // This is the actual send.
				// Our message went through, do nothing	
			}
		}
	}()

	// We return the channel and no error.
	return ch, nil
}
```

## Trying it out

To try out your new subscription visit your GraphQL playground. This is exposed on
`http://localhost:8080` by default.

Use the following query:

```graphql
subscription {
  currentTime {
    unixTime
    timeStamp
  }
}
```

Run your query and you should see a response updating with the current timestamp every
second. To gracefully stop the connection click the `Execute query` button again.


## Adding Server-Sent Events transport
You can use instead of WebSocket (or in addition) [Server-Sent Events](https://en.wikipedia.org/wiki/Server-sent_events)
as transport for subscriptions. This can have advantages and disadvantages over transport via WebSocket and requires a
compatible client library, for instance [graphql-sse](https://github.com/enisdenjo/graphql-sse). The connection between
server and client should be HTTP/2+. The client must send the subscription request via POST with
the header `accept: text/event-stream` and `content-type: application/json` in order to be accepted by the SSE transport.
The underling protocol is documented at [distinct connections mode](https://github.com/enisdenjo/graphql-sse/blob/master/PROTOCOL.md).

Add the SSE transport as first of all other transports, as the order is important. For that reason, `New` instead of
`NewDefaultServer` will be used.
```go
srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))
srv.AddTransport(transport.SSE{}) // <---- This is the important

// default server
srv.AddTransport(transport.Options{})
srv.AddTransport(transport.GET{})
srv.AddTransport(transport.POST{})
srv.AddTransport(transport.MultipartForm{})
srv.SetQueryCache(lru.New(1000))
srv.Use(extension.Introspection{})
srv.Use(extension.AutomaticPersistedQuery{
	Cache: lru.New(100),
})
```

The GraphQL playground does not support SSE yet. You can try out the subscription via curl:
```bash
curl -N --request POST --url http://localhost:8080/query \
--data '{"query":"subscription { currentTime { unixTime timeStamp } }"}' \
-H "accept: text/event-stream" -H 'content-type: application/json' \
--verbose
```

## Full Files

Here are all files at the end of this tutorial. Only files changed from the end
of the quick start are listed.

### main.go

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/example/test/graph"
	"github.com/example/test/graph/generated"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))

	srv.AddTransport(&transport.Websocket{})

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

### schema.graphqls

```graphql
type Query {
  placeholder: String
}

type Time {
  unixTime: Int!
  timeStamp: String!
}

type Subscription {
  currentTime: Time!
}
```

### schema.resolvers.go

```go
package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"time"

	"github.com/example/test/graph/generated"
	"github.com/example/test/graph/model"
)

// Placeholder is the resolver for the placeholder field.
func (r *queryResolver) Placeholder(ctx context.Context) (*string, error) {
	str := "Hello World"
	return &str, nil
}

// CurrentTime is the resolver for the currentTime field.
func (r *subscriptionResolver) CurrentTime(ctx context.Context) (<-chan *model.Time, error) {
	ch := make(chan *model.Time)

	go func() {
		defer close(ch)

		for {
			time.Sleep(1 * time.Second)
			fmt.Println("Tick")

			currentTime := time.Now()

			t := &model.Time{
				UnixTime:  int(currentTime.Unix()),
				TimeStamp: currentTime.Format(time.RFC3339),
			}

			select {
			case <-ctx.Done():
				// Exit on cancellation 
				fmt.Println("Subscription closed.")
				return
			
			case ch <- t:
				// Our message went through, do nothing
			}

		}
	}()
	return ch, nil
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation.
func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
```

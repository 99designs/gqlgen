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

1. add a WebSocket Transport to your server
2. add the `Subscription` type to your schema
3. implement a real-time resolver.

## Adding a WebSocket Transport

To send real-time data to clients, your GraphQL server needs to have an open connection
with the client. This is done using WebSockets.

To add the WebSocket transport change your `main.go` by calling `AddTransport(transport.Websocket{})`
on your query handler.

> If you are using an external router, remember to send *ALL* requests go to your handler (at `/query`),
> not just POST requests!

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

	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))

	srv.AddTransport(transport.Websocket{}) // Add WebSocket first. Here there is no config, see below for examples.
	srv.AddTransport(transport.Options{})   // If you are using the playground, it's smart to add Options and GET.
	srv.AddTransport(transport.GET{})       // ...
	srv.AddTransport(transport.POST{})      // ... Make sure this is after the WebSocket transport!

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## Configuring WebSockets

The WebSocket transport is complex, and for any non-trivial application you will need to
configure it. The transport handles this configuration by setting fields on the `transport.Websocket`
struct. For an in-depth look at all configuration options, [explore the implementation][code].

At it's most basic, the transport uses [`github.com/gorilla/websocket`][gorilla] to implement
a WebSocket connection that sets up the subscription and then sends data to the client from
the Go channel returned by the resolver. The initial handshake and the structure of the data
payloads are defined by one of two protocols: `graphql-ws` or `graphql-transport-ws` Which
one is used is negotiated by the client, defaulting to [`graphql-ws`][graphql-ws].

A minimal WebSocket configuration will handle two basic things: keep-alives and security
checks that are normally handled by HTTP middleware that may not be available or compatible
with WebSockets:

```go
srv.AddTransport(transport.Websocket{
	// Keep-alives are important for WebSockets to detect dead connections. This is
	// not unlike asking a partner who seems to have zoned out while you tell them
	// a story crucial to understanding the dynamics of your workplace: "Are you
	// listening to me?"
	//
	// Failing to set a keep-alive interval can result in the connection being held
	// open and the server expending resources to communicate with a client that has
	// long since walked to the kitchen to make a sandwich instead.
	KeepAlivePingInterval: 10 * time.Second,

	// The `github.com/gorilla/websocket.Upgrader` is used to handle the transition
	// from an HTTP connection to a WebSocket connection. Among other options, here
	// you must check the origin of the request to prevent cross-site request forgery
	// attacks.
	Upgrader: websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
				// Allow exact match on host.
				origin := r.Header.Get("Origin")
				if origin == "" || origin == r.Header.Get("Host") {
					return true
				}

				// Match on allow-listed origins.
				return slices.Contains([]string{":3000", "https://ui.mysite.com"}, origin)
		},
	},
})
```

[code]: https://github.com/99designs/gqlgen/blob/master/graphql/handler/transport/websocket.go
[gorilla]: https://pkg.go.dev/github.com/gorilla/websocket
[graphql-ws]: https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md

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

Add the SSE transport as first of all other transports, as the order is important.

```go
srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))

srv.AddTransport(transport.SSE{}) // Add SSE first.

// Continue server setup:
srv.AddTransport(transport.Options{})
srv.AddTransport(transport.GET{})
srv.AddTransport(transport.POST{})
```

Optionally add `KeepAlivePingInterval` to send a periodic heartbeat over the SSE transport.
```go
srv.AddTransport(transport.SSE{
	// Load balancers, proxies, or firewalls often have idle timeout
	// settings that specify the maximum duration a connection can
	// remain open without data being sent across it. If the idle
	// timeout is exceeded without any data being transmitted, the
	// connection may be closed when connecting SSE over HTTP/1.
	//
	// End-to-end HTTP/2 connections do not require a ping interval
	// to keep the connection open.
	KeepAlivePingInterval: 10 * time.Second,
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
	"slices"
	"time"

	"github.com/gorilla/websocket"
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

	srv := handler.New(
		generated.NewExecutableSchema(
			generated.Config{Resolvers: &graph.Resolver{}},
		),
	)
	srv.AddTransport(transport.SSE{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
					origin := r.Header.Get("Origin")
					if origin == "" || origin == r.Header.Get("Host") {
						return true
					}
					return slices.Contains([]string{":3000", "https://ui.mysite.com"}, origin)
			},
		},
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

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

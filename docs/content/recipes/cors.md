---
title: "Configure CORS for gqlgen"
description: Use a third-party CORS library (either jub0bs/cors or rs/cors) to configure CORS when working with gqlgen
linkTitle: CORS
menu: { main: { parent: "recipes" } }
---

Cross-Origin Resource Sharing (CORS) headers are required when your graphql server lives on a different domain to the one your client code is served. You can read more about CORS in the [MDN docs](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS).

gqlgen doesn't include a CORS implementation, but it is built to work with all standard http middleware. Here we are going to use the fantastic `chi` router and either `jub0bs/cors` or `rs/cors` to build our server.

## jub0bs/cors

jub0bs/cors is a user-friendly, declarative and performant CORS library for Go.

```go
package main

import (
	"net/http"
	"time"

	"github.com/99designs/gqlgen/_examples/starwars"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/jub0bs/cors"
)

func main() {
	router := chi.NewRouter()

	// Instantiate a CORS middleware.
	cors, err := cors.NewMiddleware(cors.Config{
		Origins:        []string{"http://localhost:8080"},
		Credentialed:   true,
		RequestHeaders: []string{"Accept", "Content-Type", "X-Requested-With"},
	})
	// Check for any CORS configuration mistakes.
	if err != nil {
		panic(err)
	}
	// Turn debug mode on (optional).
	cors.SetDebug(true)

	// Add the CORS middleware around every request.
	// See https://github.com/jub0bs/cors for more options.
	router.Use(cors.Wrap)

	srv := handler.New(starwars.NewExecutableSchema(starwars.NewResolver()))

	// Handle cross-origin checks in for websocket upgrade requests:
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Implementation: transport.CoderWebsocketImplementation{
			AcceptOptions: websocket.AcceptOptions{
				OriginPatterns: []string{"https://example.org"},
			},
		},
	})
	srv.AddTransport(transport.POST{})

	router.Handle("/", playground.Handler("Starwars", "/query"))
	router.Handle("/query", srv)

	err = http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}
```


## rs/cors

rs/cors is an older, imperative and flexible CORS library for Go.

```go
package main

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/_examples/starwars"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
	"github.com/coder/websocket"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	router := chi.NewRouter()

	// Add CORS middleware around every request
	// See https://github.com/rs/cors for full option listing
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8080"},
		AllowCredentials: true,
		Debug:            true,
	}).Handler)


	srv := handler.New(starwars.NewExecutableSchema(starwars.NewResolver()))

	// Handle cross-origin checks in for websocket upgrade requests:
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Implementation: transport.CoderWebsocketImplementation{
			AcceptOptions: websocket.AcceptOptions{
				OriginPatterns: []string{"https://example.org"},
			},
		},
	})
	srv.AddTransport(transport.POST{})

	router.Handle("/", playground.Handler("Starwars", "/query"))
	router.Handle("/query", srv)

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}

```

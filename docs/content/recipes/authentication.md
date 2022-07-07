---
title: "Providing authentication details through context"
description: How to using golang context.Context to authenticate users and pass user data to resolvers.
linkTitle: Authentication
menu: { main: { parent: 'recipes' } }
---

We have an app where users are authenticated using a cookie in the HTTP request, and we want to check this authentication status somewhere in our graph. Because GraphQL is transport agnostic we can't assume there will even be an HTTP request, so we need to expose these authentication details to our graph using a middleware.


```go
package auth

import (
	"database/sql"
	"net/http"
	"context"
)

// A private key for context that only this package can access. This is important
// to prevent collisions between different context uses
var userCtxKey = &contextKey{"user"}
type contextKey struct {
	name string
}

// A stand-in for our database backed user object
type User struct {
	Name string
	IsAdmin bool
}

// Middleware decodes the share session cookie and packs the session into context
func Middleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("auth-cookie")

			// Allow unauthenticated users in
			if err != nil || c == nil {
				next.ServeHTTP(w, r)
				return
			}

			userId, err := validateAndGetUserID(c)
			if err != nil {
				http.Error(w, "Invalid cookie", http.StatusForbidden)
				return
			}

			// get the user from the database
			user := getUserByID(db, userId)

			// put it in context
			ctx := context.WithValue(r.Context(), userCtxKey, user)

			// and call the next with our new context
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

// ForContext finds the user from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) *User {
	raw, _ := ctx.Value(userCtxKey).(*User)
	return raw
}
```

**Note:** `getUserByID` and `validateAndGetUserID` have been left to the user to implement.

Now when we create the server we should wrap it in our authentication middleware:
```go
package main

import (
	"net/http"

	"github.com/99designs/gqlgen/_examples/starwars"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
)

func main() {
	router := chi.NewRouter()

	router.Use(auth.Middleware(db))

	srv := handler.NewDefaultServer(starwars.NewExecutableSchema(starwars.NewResolver()))
	router.Handle("/", playground.Handler("Starwars", "/query"))
	router.Handle("/query", srv)

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}
```

And in our resolvers (or directives) we can call `ForContext` to retrieve the data back out:
```go

func (r *queryResolver) Hero(ctx context.Context, episode Episode) (Character, error) {
	if user := auth.ForContext(ctx) ; user == nil || !user.IsAdmin {
		return Character{}, fmt.Errorf("Access denied")
	}

	if episode == EpisodeEmpire {
		return r.humans["1000"], nil
	}
	return r.droid["2001"], nil
}
```

### Websockets

If you need access to the websocket init payload you can add your `InitFunc` in `AddTransport`.  
Your InitFunc implementation could then attempt to extract items from the payload:  

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/gqlgen/_examples/websocket-initfunc/server/graph"
	"github.com/gqlgen/_examples/websocket-initfunc/server/graph/generated"
	"github.com/rs/cors"
)

func webSocketInit(ctx context.Context, initPayload transport.InitPayload) (context.Context, error) {
	// Get the token from payload
	any := initPayload["authToken"]
	token, ok := any.(string)
	if !ok || token == "" {
		return nil, errors.New("authToken not found in transport payload")
	}

	// Perform token verification and authentication...
	userId := "john.doe" // e.g. userId, err := GetUserFromAuthentication(token)

	// put it in context
	ctxNew := context.WithValue(ctx, "username", userId)

	return ctxNew, nil
}

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	router := chi.NewRouter()

	// CORS setup, allow any for now
	// https://gqlgen.com/recipes/cors/
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	})

	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, error) {
			return webSocketInit(ctx, initPayload)
		},
	})
	srv.Use(extension.Introspection{})

	router.Handle("/", playground.Handler("My GraphQL App", "/app"))
	router.Handle("/app", c.Handler(srv))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
```

> Note
>
> Subscriptions are long lived, if your tokens can timeout or need to be refreshed you should keep the token in
context too and verify it is still valid in `auth.ForContext`.

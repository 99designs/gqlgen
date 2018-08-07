---
title: "Using context.Context to pass authenticated user details to resolvers"
description: How to using golang context.Context to authenticate users and pass user data to resolvers.
linkTitle: Authentication
menu: { main: { parent: 'recipes' } }
---

We have an app where users are authenticated using a cookie in the http request, and we want to check who is logged in somewhere in our graph. Because graphql is transport agnostic we cant assume there will even be a http request, so we need to build some middleware that exposes the user to our graph.


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

**note**: `getUserByID` and `validateAndGetUserID` have been left to the user to implement

Now when we create the server we should wrap it in our auth middleware:
```go
package main

import (
	"net/http"

	"github.com/99designs/gqlgen/example/starwars"
	"github.com/99designs/gqlgen/handler"
	"github.com/go-chi/chi"
)

func main() {
	router := chi.NewRouter()

	router.Use(auth.Middleware(db))

	router.Handle("/", handler.Playground("Starwars", "/query"))
	router.Handle("/query",
		handler.GraphQL(starwars.NewExecutableSchema(starwars.NewResolver())),
	)

	err := http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}
```

Now in our resolvers (and directives) we can call ForContext:
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

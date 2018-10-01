---
title: "Providing authentication details through context"
description: How to using golang context.Context to authenticate users and pass user data to resolvers.
linkTitle: Authentication
menu: { main: { parent: 'recipes' } }
---

We have an app where users are authenticated using a cookie in the HTTP request, and we want to check this authentication status somewhere in our graph. Because GraphQL is transport agnostic we can't assume there will even be an HTTP request, so we need to expose these authention details to our graph using a middleware.


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

Things are different with websockets, and if you do things in the vein of the above example, you have to compute this at every call to `auth.ForContext`.

```golang
// ForContext finds the user from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) *User {
  raw, ok := ctx.Value(userCtxKey).(*User)
  
  if !ok {
    payload := handler.GetInitPayload(ctx)
    if payload == nil {
      return nil
    }

    userId, err := validateAndGetUserID(payload["token"])
    if err != nil {
      return nil
    }

    return getUserByID(db, userId)
  }

	return raw
}
```

It's a bit inefficient if you have multiple calls to this function (e.g. on a field resolver), but what you might do to mitigate that is to have a session object set on the http request and only populate it upon the first check.
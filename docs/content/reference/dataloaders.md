---
title: "Optimizing N+1 database queries using Dataloaders"
description: Speeding up your GraphQL requests by reducing the number of round trips to the database.
linkTitle: Dataloaders
menu: { main: { parent: 'reference', weight: 10 } }
---

Dataloaders consolidate the retrieval of information into fewer, batched calls. This example demonstrates the value of dataloaders by consolidating many SQL queries into a single bulk query.

## The Problem

Imagine your graph has query that lists todos...

```graphql
query { todos { user { name } } }
```

and the `todo.user` resolver reads the `User` from a database...
```go
func (r *todoResolver) User(ctx context.Context, obj *model.Todo) (*model.User, error) {
	stmt, err := r.db.PrepareContext(ctx, "SELECT id, name FROM users WHERE id = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, obj.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, rows.Err()
	}

	var user model.User
	if err := rows.Scan(&user.ID, &user.Name); err != nil {
		return nil, err
	}
	return &user, nil
}


```

The query executor will call the `Query.Todos` resolver which does a `select * from todo` and returns `N` todos. If the nested `User` is selected, the above `UserRaw` resolver will run a separate query for each user, resulting in `N+1` database queries.

eg:
```sql
SELECT id, todo, user_id FROM todo
SELECT id, name FROM users WHERE id = ?
SELECT id, name FROM users WHERE id = ?
SELECT id, name FROM users WHERE id = ?
SELECT id, name FROM users WHERE id = ?
SELECT id, name FROM users WHERE id = ?
SELECT id, name FROM users WHERE id = ?
```

Whats even worse? most of those todos are all owned by the same user! We can do better than this.

## Dataloader

Dataloaders allow us to consolidate the fetching of `todo.user` across all resolvers for a given GraphQL request into a single database query and even cache the results for subsequent requests.

We're going to use [graph-gophers/dataloader](https://github.com/graph-gophers/dataloader) to implement a dataloader for bulk-fetching users.

```bash
go get -u github.com/graph-gophers/dataloader/v7
```

Next, we implement a data loader and a middleware for injecting the data loader on a request context.

```go
package loaders

// import graph gophers with your other imports
import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/graph-gophers/dataloader/v7"
)

type ctxKey string

const (
	loadersKey = ctxKey("dataloaders")
)

// userReader reads Users from a database
type userReader struct {
	db *sql.DB
}

// getUsers implements a batch function that can retrieve many users by ID,
// for use in a dataloader
func (u *userReader) getUsers(ctx context.Context, userIds []string) []*dataloader.Result[*model.User] {
	stmt, err := u.db.PrepareContext(ctx, `SELECT id, name FROM users WHERE id IN (?`+strings.Repeat(",?", len(userIds)-1)+`)`)
	if err != nil {
		return handleError[*model.User](len(userIds), err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, userIds)
	if err != nil {
		return handleError[*model.User](len(userIds), err)
	}
	defer rows.Close()

	result := make([]*dataloader.Result[*model.User], 0, len(userIds))
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Name); err != nil {
			result = append(result, &dataloader.Result[*model.User]{Error: err})
			continue
		}
		result = append(result, &dataloader.Result[*model.User]{Data: &user})
	}
	return result
}

// handleError creates array of result with the same error repeated for as many items requested
func handleError[T any](itemsLength int, err error) []*dataloader.Result[T] {
	result := make([]*dataloader.Result[T], itemsLength)
	for i := 0; i < itemsLength; i++ {
		result[i] = &dataloader.Result[T]{Error: err}
	}
	return result
}

// Loaders wrap your data loaders to inject via middleware
type Loaders struct {
	UserLoader *dataloader.Loader[string, *model.User]
}

// NewLoaders instantiates data loaders for the middleware
func NewLoaders(conn *sql.DB) *Loaders {
	// define the data loader
	ur := &userReader{db: conn}
	return &Loaders{
		UserLoader: dataloader.NewBatchedLoader(ur.getUsers, dataloader.WithWait[string, *model.User](time.Millisecond)),
	}
}

// Middleware injects data loaders into the context
func Middleware(conn *sql.DB, next http.Handler) http.Handler {
	// return a middleware that injects the loader to the request context
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loader := NewLoaders(conn)
		r = r.WithContext(context.WithValue(r.Context(), loadersKey, loader))
		next.ServeHTTP(w, r)
	})
}

// For returns the dataloader for a given context
func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

// GetUser returns single user by id efficiently
func GetUser(ctx context.Context, userID string) (*model.User, error) {
	loaders := For(ctx)
	return loaders.UserLoader.Load(ctx, userID)()
}

// GetUsers returns many users by ids efficiently
func GetUsers(ctx context.Context, userIDs []string) ([]*model.User, []error) {
	loaders := For(ctx)
	return loaders.UserLoader.LoadMany(ctx, userIDs)()
}

```

Add the dataloader middleware to your server...
```go
// create the query handler
var srv http.Handler = handler.NewDefaultServer(generated.NewExecutableSchema(...))
// wrap the query handler with middleware to inject dataloader in requests.
// pass in your dataloader dependencies, in this case the db connection.
srv = loaders.Middleware(db, srv)
// register the wrapped handler
http.Handle("/query", srv)
```

Now lets update our resolver to call the dataloader:
```go
func (r *todoResolver) User(ctx context.Context, obj *model.Todo) (*model.User, error) {
	return loaders.GetUser(ctx, obj.UserID)
}
```

The end result? Just 2 queries!
```sql
SELECT id, todo, user_id FROM todo
SELECT id, name from user WHERE id IN (?,?,?,?,?)
```

You can see an end-to-end example [here](https://github.com/zenyui/gqlgen-dataloader).

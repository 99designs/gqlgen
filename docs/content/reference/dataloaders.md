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

We're going to use [vikstrous/dataloadgen](https://github.com/vikstrous/dataloadgen) to implement a dataloader for bulk-fetching users.

```bash
go get github.com/vikstrous/dataloadgen
```

Next, we implement a data loader and a middleware for injecting the data loader on a request context.

```go
package loader

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/vikstrous/dataloadgen"
)

// Get returns the Loaders bundle from the context. It must be used only in XXXXXXXXXXXX resolvers where Middleware has put the Loaders struct into the context already.
func Get(ctx context.Context) *Loaders {
	return ctx.Value(ctxKey{}).(*Loaders)
}


// Loaders provide access for loading various objects from the underlying object's storage system while batching concurrent requests and caching responses.
type Loaders struct {
	User *dataloadgen.Loader[string, *model.User]
}

// Middleware injects data loaders into the context
func Middleware(conn *sql.DB, next http.Handler) http.Handler {
	// return a middleware that injects the loader to the request context
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Note that the loaders are being created per-request. This is important because they contain caching and batching logic that must be request-scoped.
		loaders := newLoaders(conn)
		r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, loaders))
		next.ServeHTTP(w, r)
	})
}

type ctxKey struct{}

// newLoaders creates the Loaders struct
func newLoaders(conn *sql.DB) *Loaders {
	ur := &userFetcher{db: conn}
	return &Loaders{
		User: dataloadgen.NewLoader(ur.getUsers, dataloadgen.WithWait(time.Millisecond)),
	}
}

// userFetcher reads Users from a database
type userFetcher struct {
	db *sql.DB
}

// getUsers implements a batch function that can retrieve many users by ID,
// for use in a dataloader
func (u *userFetcher) getUsers(ctx context.Context, userIDs []string) ([]*model.User, []error) {
	stmt, err := u.db.PrepareContext(ctx, `SELECT id, name FROM users WHERE id IN (?`+strings.Repeat(",?", len(userIDs)-1)+`)`)
	if err != nil {
		return nil, []error{err}
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, userIDs)
	if err != nil {
		return nil, []error{err}
	}
	defer rows.Close()

	users := make([]*model.User, 0, len(userIDs))
	errs := make([]error, 0, len(userIDs))
	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.ID, &user.Name); err != nil {
			errs = append(errs, err)
			continue
		}
		users = append(users, &user)
	}
	return users, errs
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
	return loaders.Get(ctx).User.Load(ctx, obj.UserID)
}
```

The end result? Just 2 queries!
```sql
SELECT id, todo, user_id FROM todo
SELECT id, name from user WHERE id IN (?,?,?,?,?)
```

You can see an end-to-end example [here](https://github.com/vikstrous/dataloadgen-example).
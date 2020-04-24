---
title: "Optimizing N+1 database queries using Dataloaders"
description: Speeding up your GraphQL requests by reducing the number of round trips to the database.
linkTitle: Dataloaders
menu: { main: { parent: 'reference', weight: 10 } }
---

Have you noticed some GraphQL queries end can make hundreds of database
queries, often with mostly repeated data? Lets take a look why and how to
fix it.

## Query Resolution

Imagine if you had a simple query like this:

```graphql
query { todos { users { name } } }
```

and our todo.user resolver looks like this:
```go
func (r *todoResolver) UserRaw(ctx context.Context, obj *model.Todo) (*model.User, error) {
	res := db.LogAndQuery(r.Conn, "SELECT id, name FROM dataloader_example.user WHERE id = ?", obj.UserID)
	defer res.Close()

	if !res.Next() {
		return nil, nil
	}
	var user model.User
	if err := res.Scan(&user.ID, &user.Name); err != nil {
		panic(err)
	}
	return &user, nil
}
```

**Note**: I'm going to use go's low level `sql.DB` here. All of this will
work with whatever your favourite ORM is.

The query executor will call the Query.Todos resolver which does a `select * from todo` and
return N todos. Then for each of the todos, concurrently, call the Todo_user resolver,
`SELECT from USER where id = todo.user_id`.


eg:
```sql
SELECT id, todo, user_id FROM todo
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
SELECT id, name FROM user WHERE id = ?
```

Whats even worse? most of those todos are all owned by the same user! We can do better than this.

## Dataloader

What we need is a way to group up all of those concurrent requests, take out any duplicates, and
store them in case they are needed later on in request. The dataloader is just that, a request-scoped
batching and caching solution popularised by [facebook](https://github.com/facebook/dataloader).

We're going to use [dataloaden](https://github.com/vektah/dataloaden) to build our dataloaders.
In languages with generics, we could probably just create a DataLoader<User>, but golang
doesnt have generics. Instead we generate the code manually for our instance.

```bash
go get github.com/vektah/dataloaden
mkdir dataloader
cd dataloader
go run github.com/vektah/dataloaden UserLoader int *gqlgen-tutorials/dataloader/graph/model.User
```

Next we need to create an instance of our new dataloader and tell how to fetch data.
Because dataloaders are request scoped, they are a good fit for `context`.

```go

const loadersKey = "dataloaders"

type Loaders struct {
	UserById UserLoader
}

func Middleware(conn *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loadersKey, &Loaders{
			UserById: UserLoader{
				maxBatch: 100,
				wait:     1 * time.Millisecond,
				fetch: func(ids []int) ([]*model.User, []error) {
					placeholders := make([]string, len(ids))
					args := make([]interface{}, len(ids))
					for i := 0; i < len(ids); i++ {
						placeholders[i] = "?"
						args[i] = i
					}

					res := db.LogAndQuery(conn,
						"SELECT id, name from dataloader_example.user WHERE id IN ("+strings.Join(placeholders, ",")+")",
						args...,
					)
					defer res.Close()

					userById := map[int]*model.User{}
					for res.Next() {
						user := model.User{}
						err := res.Scan(&user.ID, &user.Name)
						if err != nil {
							panic(err)
						}
						userById[user.ID] = &user
					}

					users := make([]*model.User, len(ids))
					for i, id := range ids {
						users[i] = userById[id]
						i++
					}

					return users, nil
				},
			},
		})
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

```

This dataloader will wait for up to 1 millisecond to get 100 unique requests and then call
the fetch function. This function is a little ugly, but half of it is just building the SQL!

Now lets update our resolver to call the dataloader:
```go
func (r *todoResolver) UserLoader(ctx context.Context, obj *model.Todo) (*model.User, error) {
	return dataloader.For(ctx).UserById.Load(obj.UserID)
}
```

The end result? just 2 queries!
```sql
SELECT id, todo, user_id FROM todo
SELECT id, name from user WHERE id IN (?,?,?,?,?)
```

The generated UserLoader has a few other useful methods on it:

 - `LoadAll(keys)`: If you know up front you want a bunch users
 - `Prime(key, user)`: Used to sync state between similar loaders (usersById, usersByNote)

You can see the full working example [here](https://github.com/vektah/gqlgen-tutorials/tree/master/dataloader).

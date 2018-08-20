---
title: "Optimizing N+1 database queries using Dataloaders"
description: Speeding up your GraphQL requests by reducing the number of round trips to the database.  
linkTitle: Dataloaders
menu: { main: { parent: 'reference' } }
---

Have you noticed some GraphQL queries end can make hundreds of database 
queries, often with mostly repeated data? Lets take a look why and how to 
fix it.  

## Query Resolution

Imagine if you had a simple query like this:

```graphql
query { todos { users { name } }
```

and our todo.user resolver looks like this:
```go
func (r *Resolver) Todo_user(ctx context.Context, obj *Todo) (*User, error) {
	res := logAndQuery(r.db, "SELECT id, name FROM user WHERE id = ?", obj.UserID)
	defer res.Close()

	if !res.Next() {
		return nil, nil
	}
	var user User
	if err := res.Scan(&user.ID, &user.Name); err != nil {
		panic(err)
	}
	return &user, nil
}
```

**Note**: I'm going to use go's low level `sql.DB` here. All of this will 
work with whatever your favourite ORM is.

The query executor will call the Query_todos resolver which does a `select * from todo` and 
return N todos. Then for each of the todos, concurrently, call the Todo_user resolver,
`SELECT from USER where id = todo.id`.


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

dataloaden github.com/full/package/name.User
```

Next we need to create an instance of our new dataloader and tell how to fetch data. 
Because dataloaders are request scoped, they are a good fit for `context`.

```go

const userLoaderKey = "userloader"

func DataloaderMiddleware(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userloader := UserLoader{
			maxBatch: 100,
			wait:     1 * time.Millisecond,
			fetch: func(ids []int) ([]*User, []error) {
				placeholders := make([]string, len(ids))
				args := make([]interface{}, len(ids))
				for i := 0; i < len(ids); i++ {
					placeholders[i] = "?"
					args[i] = i
				}

				res := logAndQuery(db,
					"SELECT id, name from user WHERE id IN ("+
						strings.Join(placeholders, ",")+")",
					args...,
				)
				
				defer res.Close()

				users := make([]*User, len(ids))
				i := 0
				for res.Next() {
					users[i] = &User{}
					err := res.Scan(&users[i].ID, &users[i].Name)
					if err != nil {
						panic(err)
					}
					i++
				}

				return users, nil
			},
		}
		ctx := context.WithValue(r.Context(), userLoaderKey, &userloader)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func (r *Resolver) Todo_userLoader(ctx context.Context, obj *Todo) (*User, error) {
	return ctx.Value(userLoaderKey).(*UserLoader).Load(obj.UserID)
}
```  

This dataloader will wait for up to 1 millisecond to get 100 unique requests and then call 
the fetch function. This function is a little ugly, but half of it is just building the SQL!

The end result? just 2 queries!
```sql
SELECT id, todo, user_id FROM todo
SELECT id, name from user WHERE id IN (?,?,?,?,?)
```

The generated UserLoader has a few other useful methods on it:

 - `LoadAll(keys)`: If you know up front you want a bunch users
 - `Prime(key, user)`: Used to sync state between similar loaders (usersById, usersByNote)

You can see the full working example [here](https://github.com/vektah/gqlgen-tutorials/tree/master/dataloader)

package singlefile

import (
	"context"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.ErrorBubble = func(ctx context.Context) (i *Error, e error) {
		return &Error{ID: "E1234"}, nil
	}

	resolvers.QueryResolver.User = func(ctx context.Context, id int) (user *User, e error) {
		return &User{ID: 1}, nil
	}

	resolvers.UserResolver.Friends = func(ctx context.Context, obj *User) (users []*User, e error) {
		return []*User{{ID: 1}}, nil
	}

	resolvers.UserResolver.Pets = func(ctx context.Context, obj *User, limit *int) ([]*Pet, error) {
		return []*Pet{{ID: 10}}, nil
	}

	resolvers.PetResolver.Friends = func(ctx context.Context, obj *Pet, limit *int) ([]*Pet, error) {
		return []*Pet{}, nil
	}

	resolvers.QueryResolver.ModelMethods = func(ctx context.Context) (methods *ModelMethods, e error) {
		return &ModelMethods{}, nil
	}

	var mu sync.Mutex
	areMethods := map[string]bool{}
	areResolvers := map[string]bool{}
	srv := handler.NewDefaultServer(
		NewExecutableSchema(Config{Resolvers: resolvers}),
	)
	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		path, _ := ctx.Value(ckey("path")).([]int)
		return next(context.WithValue(ctx, ckey("path"), append(path, 1)))
	})

	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		path, _ := ctx.Value(ckey("path")).([]int)
		return next(context.WithValue(ctx, ckey("path"), append(path, 2)))
	})

	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		fc := graphql.GetFieldContext(ctx)
		mu.Lock()
		areMethods[fc.Field.Name] = fc.IsMethod
		areResolvers[fc.Field.Name] = fc.IsResolver
		mu.Unlock()
		return next(ctx)
	})

	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		fc := graphql.GetFieldContext(ctx)
		if fc.Field.Name != "user" {
			return next(ctx)
		}
		opCtx := graphql.GetOperationContext(ctx)
		collected := graphql.CollectFields(opCtx, fc.Field.Selections, []string{"User"})
		require.Len(t, collected, 3)
		require.Equal(t, "pets", collected[2].Name)
		child, err := fc.Child(ctx, collected[2])
		require.NoError(t, err)
		require.Equal(t, 2, *child.Args["limit"].(*int))
		collected = graphql.CollectFields(opCtx, child.Field.Selections, []string{"Pet"})
		require.Len(t, collected, 2)
		require.Equal(t, "friends", collected[1].Name)
		child, err = child.Child(ctx, collected[1])
		require.NoError(t, err)
		require.Equal(t, 10, *child.Args["limit"].(*int))
		return next(ctx)
	})

	c := client.New(srv)

	var resp struct {
		User struct {
			ID      int
			Friends []struct {
				ID int
			}
			Pets []struct {
				ID      int
				Friends []struct {
					ID int
				}
			}
		}
		ModelMethods struct {
			NoContext bool
		}
	}

	called := false
	resolvers.UserResolver.Friends = func(ctx context.Context, obj *User) ([]*User, error) {
		assert.Equal(t, []int{1, 2, 1, 2}, ctx.Value(ckey("path")))
		called = true
		return []*User{}, nil
	}

	err := c.Post(`query {
		user(id: 1) {
			id,
			friends {
				id
			}
			pets (limit: 2) {
				id
				friends(limit: 10) {
					id
				}
			}
		}
		modelMethods {
			noContext
		}
	}`, &resp)

	assert.Equal(t, map[string]bool{
		"user":         true,
		"pets":         true,
		"id":           false,
		"friends":      true,
		"modelMethods": true,
		"noContext":    true,
	}, areMethods)
	assert.Equal(t, map[string]bool{
		"user":         true,
		"pets":         true,
		"id":           false,
		"friends":      true,
		"modelMethods": true,
		"noContext":    false,
	}, areResolvers)

	require.NoError(t, err)
	require.True(t, called)
}

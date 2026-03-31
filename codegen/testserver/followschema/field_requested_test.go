package followschema

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

func TestFieldRequested(t *testing.T) {
	resolvers := &Stub{}

	srv := handler.New(NewExecutableSchema(Config{Resolvers: resolvers}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	var friendsRequested, petsRequested bool
	var nestedPath string
	var nestedResult bool
	var anyResult bool
	var anyPaths []string

	resolvers.QueryResolver.User = func(ctx context.Context, id int) (*User, error) {
		friendsRequested = graphql.FieldRequested(ctx, "friends")
		petsRequested = graphql.FieldRequested(ctx, "pets")
		if nestedPath != "" {
			nestedResult = graphql.FieldRequested(ctx, nestedPath)
		}
		if len(anyPaths) > 0 {
			anyResult = graphql.AnyFieldRequested(ctx, anyPaths...)
		}
		return &User{
			ID:      id,
			Created: time.Now(),
		}, nil
	}

	resolvers.UserResolver.Friends = func(ctx context.Context, obj *User) ([]*User, error) {
		return []*User{{ID: 2, Created: time.Now()}}, nil
	}

	resolvers.UserResolver.Pets = func(ctx context.Context, obj *User, limit *int) ([]*Pet, error) {
		return []*Pet{{ID: 10}}, nil
	}

	resolvers.PetResolver.Friends = func(ctx context.Context, obj *Pet, limit *int) ([]*Pet, error) {
		return nil, nil
	}

	t.Run("friends not requested", func(t *testing.T) {
		friendsRequested = false
		nestedPath = ""
		var resp struct {
			User struct {
				ID int `json:"id"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { id } }`, &resp)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.User.ID)
		assert.False(t, friendsRequested)
	})

	t.Run("friends requested", func(t *testing.T) {
		friendsRequested = false
		nestedPath = ""
		var resp struct {
			User struct {
				ID      int `json:"id"`
				Friends []struct {
					ID int `json:"id"`
				} `json:"friends"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { id friends { id } } }`, &resp)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.User.ID)
		assert.True(t, friendsRequested)
		require.Len(t, resp.User.Friends, 1)
		assert.Equal(t, 2, resp.User.Friends[0].ID)
	})

	t.Run("multiple fields - friends and pets", func(t *testing.T) {
		friendsRequested = false
		petsRequested = false
		nestedPath = ""
		var resp struct {
			User struct {
				ID      int `json:"id"`
				Friends []struct {
					ID int `json:"id"`
				} `json:"friends"`
				Pets []struct {
					ID int `json:"id"`
				} `json:"pets"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { id friends { id } pets { id } } }`, &resp)
		require.NoError(t, err)
		assert.True(t, friendsRequested)
		assert.True(t, petsRequested)
	})

	t.Run("pets requested but not friends", func(t *testing.T) {
		friendsRequested = false
		petsRequested = false
		nestedPath = ""
		var resp struct {
			User struct {
				ID   int `json:"id"`
				Pets []struct {
					ID int `json:"id"`
				} `json:"pets"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { id pets { id } } }`, &resp)
		require.NoError(t, err)
		assert.False(t, friendsRequested)
		assert.True(t, petsRequested)
	})

	t.Run("nested path - friends.id", func(t *testing.T) {
		nestedPath = "friends.id"
		nestedResult = false
		var resp struct {
			User struct {
				Friends []struct {
					ID int `json:"id"`
				} `json:"friends"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { friends { id } } }`, &resp)
		require.NoError(t, err)
		assert.True(t, nestedResult)
	})

	t.Run("nested path - missing leaf", func(t *testing.T) {
		nestedPath = "friends.created"
		nestedResult = true
		var resp struct {
			User struct {
				Friends []struct {
					ID int `json:"id"`
				} `json:"friends"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { friends { id } } }`, &resp)
		require.NoError(t, err)
		assert.False(t, nestedResult, "friends.created should be false when only friends.id is selected")
	})

	t.Run("inline fragment", func(t *testing.T) {
		friendsRequested = false
		nestedPath = ""
		var resp struct {
			User struct {
				ID      int `json:"id"`
				Friends []struct {
					ID int `json:"id"`
				} `json:"friends"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { id ... on User { friends { id } } } }`, &resp)
		require.NoError(t, err)
		assert.True(t, friendsRequested, "friends should be detected inside inline fragment")
	})

	t.Run("named fragment", func(t *testing.T) {
		friendsRequested = false
		nestedPath = ""
		var resp struct {
			User struct {
				ID      int `json:"id"`
				Friends []struct {
					ID int `json:"id"`
				} `json:"friends"`
			} `json:"user"`
		}
		err := c.Post(`
			fragment UserFriends on User { friends { id } }
			query { user(id: 1) { id ...UserFriends } }
		`, &resp)
		require.NoError(t, err)
		assert.True(t, friendsRequested, "friends should be detected inside named fragment")
	})

	t.Run("skip directive - friends skipped", func(t *testing.T) {
		friendsRequested = false
		nestedPath = ""
		var resp struct {
			User struct {
				ID int `json:"id"`
			} `json:"user"`
		}
		err := c.Post(
			`query ($skip: Boolean!) { user(id: 1) { id friends @skip(if: $skip) { id } } }`,
			&resp,
			client.Var("skip", true),
		)
		require.NoError(t, err)
		assert.False(t, friendsRequested, "friends should not be requested when @skip(if: true)")
	})

	t.Run("skip directive - friends not skipped", func(t *testing.T) {
		friendsRequested = false
		nestedPath = ""
		var resp struct {
			User struct {
				ID      int `json:"id"`
				Friends []struct {
					ID int `json:"id"`
				} `json:"friends"`
			} `json:"user"`
		}
		err := c.Post(
			`query ($skip: Boolean!) { user(id: 1) { id friends @skip(if: $skip) { id } } }`,
			&resp,
			client.Var("skip", false),
		)
		require.NoError(t, err)
		assert.True(t, friendsRequested, "friends should be requested when @skip(if: false)")
	})

	t.Run("include directive - friends excluded", func(t *testing.T) {
		friendsRequested = false
		nestedPath = ""
		var resp struct {
			User struct {
				ID int `json:"id"`
			} `json:"user"`
		}
		err := c.Post(
			`query ($inc: Boolean!) { user(id: 1) { id friends @include(if: $inc) { id } } }`,
			&resp,
			client.Var("inc", false),
		)
		require.NoError(t, err)
		assert.False(t, friendsRequested, "friends should not be requested when @include(if: false)")
	})

	t.Run("field that does not exist in selection", func(t *testing.T) {
		nestedPath = "nonexistent"
		nestedResult = true
		var resp struct {
			User struct {
				ID int `json:"id"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { id } }`, &resp)
		require.NoError(t, err)
		assert.False(t, nestedResult, "nonexistent field should return false")
	})

	t.Run("aliased field", func(t *testing.T) {
		friendsRequested = false
		nestedPath = ""
		var resp struct {
			User struct {
				ID    int `json:"id"`
				Mates []struct {
					ID int `json:"id"`
				} `json:"mates"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { id mates: friends { id } } }`, &resp)
		require.NoError(t, err)
		assert.True(t, friendsRequested, "aliased field should still be detected by its original name")
	})

	t.Run("AnyFieldRequested - one matches", func(t *testing.T) {
		anyPaths = []string{"friends", "nonexistent", "alsoMissing"}
		anyResult = false
		var resp struct {
			User struct {
				ID      int `json:"id"`
				Friends []struct {
					ID int `json:"id"`
				} `json:"friends"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { id friends { id } } }`, &resp)
		require.NoError(t, err)
		assert.True(t, anyResult, "should be true when at least one field matches")
	})

	t.Run("AnyFieldRequested - none match", func(t *testing.T) {
		anyPaths = []string{"nonexistent", "alsoMissing"}
		anyResult = true
		var resp struct {
			User struct {
				ID int `json:"id"`
			} `json:"user"`
		}
		err := c.Post(`{ user(id: 1) { id } }`, &resp)
		require.NoError(t, err)
		assert.False(t, anyResult, "should be false when no fields match")
	})
}

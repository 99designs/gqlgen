//go:generate rm -f resolver.go
//go:generate go run ../../../testdata/gqlgen.go -config gqlgen.yml -stub stub.go

package usefunctionsyntaxforexecutioncontext

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql/handler"
)

func TestQuery(t *testing.T) {
	resolvers := &Stub{}
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			Log: LogDirective,
		},
	}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	testAge := new(int)
	createdAt := "2021-01-01"
	resolvers.QueryResolver.GetUser = func(ctx context.Context, id string) (*User, error) {
		return &User{
			ID:        id,
			Name:      "test",
			Email:     "testEmail",
			Age:       testAge,
			Role:      "ADMIN",
			CreatedAt: &createdAt,
		}, nil
	}

	resolvers.QueryResolver.ListUsers = func(ctx context.Context, filter *UserFilter) ([]*User, error) {
		return []*User{
			{
				ID:        "1",
				Name:      "test1",
				Email:     "testEmail",
				Age:       testAge,
				Role:      "ADMIN",
				CreatedAt: &createdAt,
			},
			{
				ID:        "2",
				Name:      "test2",
				Email:     "testEmail",
				Age:       testAge,
				Role:      "ADMIN",
				CreatedAt: &createdAt,
			},
		}, nil
	}

	expectedJsonResp := `
	{
		"getUser": {
			"id": "1",
			"name": "test",
			"email": "testEmail",
			"age": 0,
			"role": "ADMIN",
			"createdAt": "2021-01-01"
		},
		"listUsers": [
			{
			"id": "1",
			"name": "test1",
			"email": "testEmail",
			"age": 0,
			"role": "ADMIN",
			"createdAt": "2021-01-01"
			},
			{
			"id": "2",
			"name": "test2",
			"email": "testEmail",
			"age": 0,
			"role": "ADMIN",
			"createdAt": "2021-01-01"
			}
		]
	}
	`

	t.Run("test query", func(t *testing.T) {
		var resp struct {
			GetUser struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Email     string `json:"email"`
				Age       *int   `json:"age"`
				Role      string `json:"role"`
				CreatedAt string `json:"createdAt"`
			} `json:"getUser"`

			ListUsers []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Email     string `json:"email"`
				Age       *int   `json:"age"`
				Role      string `json:"role"`
				CreatedAt string `json:"createdAt"`
			} `json:"listUsers"`
		}
		c.MustPost(`query TestQuery {
			# Fetch a user by ID
			getUser(id: "1") {
				id
				name
				email
				age
				role
				createdAt
			}

			# List users with a filter
			listUsers(filter: { isActive: true, roles: [ADMIN, USER] }) {
				id
				name
				email
				age
				role
				createdAt
			}
			}
		`, &resp)
		jsonResp, err := json.Marshal(resp)
		require.NoError(t, err)
		require.JSONEq(t, expectedJsonResp, string(jsonResp))
	})
}

func TestMutation(t *testing.T) {
	resolvers := &Stub{}
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			Log: LogDirective,
		},
	}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	createdAt := "2021-01-01"
	resolvers.MutationResolver.CreateUser = func(ctx context.Context, input CreateUserInput) (*User, error) {
		return &User{
			ID:        "1",
			Name:      input.Name,
			Email:     input.Email,
			Age:       input.Age,
			Role:      *input.Role,
			CreatedAt: &createdAt,
		}, nil
	}

	message := "User deleted successfully"
	resolvers.MutationResolver.DeleteUser = func(ctx context.Context, id string) (*MutationResponse, error) {
		return &MutationResponse{
			Success: true,
			Message: &message,
		}, nil
	}

	expectedJsonResp := `
	{
		"createUser": {
			"id": "1",
			"name": "test",
			"email": "testEmail",
			"age": 0,
			"role": "ADMIN",
			"createdAt": "2021-01-01"
		},
		"deleteUser": {
			"success": true,
			"message": "User deleted successfully"
		}
	}
	`

	t.Run("test mutation", func(t *testing.T) {
		var resp struct {
			CreateUser struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Email     string `json:"email"`
				Age       *int   `json:"age"`
				Role      string `json:"role"`
				CreatedAt string `json:"createdAt"`
			} `json:"createUser"`

			DeleteUser struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			} `json:"deleteUser"`
		}

		c.MustPost(`mutation TestMutation {
			createUser(input: { name: "test", email: "testEmail", age: 0, role: ADMIN }) {
				id
				name
				email
				age
				role
				createdAt
			}
			deleteUser(id: "1") {
				success
				message
			}
		}`, &resp)

		jsonResp, err := json.Marshal(resp)
		require.NoError(t, err)
		require.JSONEq(t, expectedJsonResp, string(jsonResp))
	})
}

func TestSubscription(t *testing.T) {
	resolvers := &Stub{}
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			Log: LogDirective,
		},
	}))
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: time.Second,
	})
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	createdAt := "2021-01-01"
	resolvers.SubscriptionResolver.UserCreated = func(ctx context.Context) (<-chan *User, error) {
		ch := make(chan *User, 1)
		go func() {
			defer close(ch)
			ch <- &User{
				ID:        "1",
				Name:      "testUser",
				Email:     "testEmail",
				Age:       nil,
				Role:      "ADMIN",
				CreatedAt: &createdAt,
			}
		}()
		return ch, nil
	}

	t.Run("test subscription", func(t *testing.T) {
		var resp struct {
			UserCreated struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Email     string `json:"email"`
				Age       *int   `json:"age"`
				Role      string `json:"role"`
				CreatedAt string `json:"createdAt"`
			} `json:"userCreated"`
		}

		expectedJsonResp := `
		{
			"userCreated": {
				"id": "1",
				"name": "testUser",
				"email": "testEmail",
				"age": null,
				"role": "ADMIN",
				"createdAt": "2021-01-01"
			}
		}
		`

		sub := c.Websocket(`subscription TestSubscription {
			userCreated {
				id
				name
				email
				age
				role
				createdAt
			}
		}`)

		defer sub.Close()

		err := sub.Next(&resp)
		require.NoError(t, err)

		jsonResp, err := json.Marshal(resp)
		require.NoError(t, err)
		require.JSONEq(t, expectedJsonResp, string(jsonResp))
	})
}

package introspection

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/ast"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
)

const userCtxKey = "user_key"

type user struct {
	ID string
}

func getUserFromContext(ctx context.Context) *user {
	if v := ctx.Value(userCtxKey); v != nil {
		return v.(*user)
	}
	return nil
}

func newClient(injectUser *user) *client.Client {
	return client.New(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if injectUser != nil {
				ctx := context.WithValue(r.Context(), userCtxKey, injectUser)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}(handler.GraphQL(NewExecutableSchema(Config{
		Directives: DirectiveRoot{
			Hide: func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
				return next(ctx)
			},
			Introspection: IntrospectionDirective(IntrospectionConfig{
				HideFunc: func(ctx context.Context, directive *ast.Directive) (allow bool, err error) {
					return false, nil
				},
				RequireAuthFunc: func(ctx context.Context, directive *ast.Directive) (allow bool, err error) {
					u := getUserFromContext(ctx)
					return u != nil, nil
				},
				RequireOwnerFunc: func(ctx context.Context, directive *ast.Directive) (allow bool, err error) {
					u := getUserFromContext(ctx)
					if u != nil {
						return u.ID == "1", nil
					}
					return false, nil
				},
			}),
			RequireAuth: func(ctx context.Context, obj interface{}, next graphql.Resolver, roles []Role) (res interface{}, err error) {
				return next(ctx)
			},
			RequireOwner: func(ctx context.Context, obj interface{}, next graphql.Resolver) (res interface{}, err error) {
				return next(ctx)
			},
		},
	}))))
}

type responseObject struct {
	Type struct {
		Fields []responseField `json:"fields"`
	} `json:"__type"`
}

type responseInputObject struct {
	Type struct {
		Fields []responseField `json:"inputFields"`
	} `json:"__type"`
}

type responseField struct {
	Name string `json:"name"`
}

func TestIntrospection(t *testing.T) {

	t.Run("disallow object field", func(t *testing.T) {
		c := newClient(nil)

		query := `{
			__type(name: "User") {
    			fields {
      				name
    			}
  			}
		}`
		var resp responseObject
		c.MustPost(query, &resp)
		require.Contains(t, resp.Type.Fields, responseField{
			Name: "id",
		})
		require.NotContains(t, resp.Type.Fields, responseField{
			Name: "email",
		})
		require.NotContains(t, resp.Type.Fields, responseField{
			Name: "passwordHash",
		})
	})

	t.Run("allow object field", func(t *testing.T) {
		c := newClient(&user{
			ID: "1",
		})

		query := `{
			__type(name: "User") {
    			fields {
      				name
    			}
  			}
		}`
		var resp responseObject
		c.MustPost(query, &resp)
		require.Contains(t, resp.Type.Fields, responseField{
			Name: "id",
		})
		require.Contains(t, resp.Type.Fields, responseField{
			Name: "email",
		})
		require.NotContains(t, resp.Type.Fields, responseField{
			Name: "passwordHash",
		})
	})

	t.Run("disallow input field", func(t *testing.T) {
		c := newClient(&user{
			ID: "5",
		})

		query := `{
			__type(name: "UserUpdateInput") {
    			inputFields {
      				name
    			}
  			}
		}`
		var resp responseInputObject
		c.MustPost(query, &resp)
		require.Contains(t, resp.Type.Fields, responseField{
			Name: "id",
		})
		require.NotContains(t, resp.Type.Fields, responseField{
			Name: "email",
		})
		require.NotContains(t, resp.Type.Fields, responseField{
			Name: "password",
		})
	})

	t.Run("allow input field", func(t *testing.T) {
		c := newClient(&user{
			ID: "1",
		})

		query := `{
			__type(name: "UserUpdateInput") {
    			inputFields {
      				name
    			}
  			}
		}`
		var resp responseInputObject
		c.MustPost(query, &resp)
		require.Contains(t, resp.Type.Fields, responseField{
			Name: "id",
		})
		require.Contains(t, resp.Type.Fields, responseField{
			Name: "email",
		})
		require.Contains(t, resp.Type.Fields, responseField{
			Name: "password",
		})
	})
}

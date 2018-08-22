//go:generate rm -f resolver.go
//go:generate gorunpkg github.com/99designs/gqlgen

package testserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestGeneratedResolversAreValid(t *testing.T) {
	http.Handle("/query", handler.GraphQL(NewExecutableSchema(Config{
		Resolvers: &Resolver{},
	})))
}

func TestForcedResolverFieldIsPointer(t *testing.T) {
	field, ok := reflect.TypeOf((*ForcedResolverResolver)(nil)).Elem().MethodByName("Field")
	require.True(t, ok)
	require.Equal(t, "*testserver.Circle", field.Type.Out(0).String())
}

func TestGeneratedServer(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{
		Resolvers: &testResolver{},
	})))
	c := client.New(srv.URL)

	t.Run("null bubbling", func(t *testing.T) {
		t.Run("when function errors on non required field", func(t *testing.T) {
			var resp struct {
				Valid       string
				ErrorBubble *struct {
					Id                      string
					ErrorOnNonRequiredField *string
				}
			}
			err := c.Post(`query { valid, errorBubble { id, errorOnNonRequiredField } }`, &resp)

			require.EqualError(t, err, `[{"message":"boom","path":["errorBubble","errorOnNonRequiredField"]}]`)
			require.Equal(t, "E1234", resp.ErrorBubble.Id)
			require.Nil(t, resp.ErrorBubble.ErrorOnNonRequiredField)
			require.Equal(t, "Ok", resp.Valid)
		})

		t.Run("when function errors", func(t *testing.T) {
			var resp struct {
				Valid       string
				ErrorBubble *struct {
					NilOnRequiredField string
				}
			}
			err := c.Post(`query { valid, errorBubble { id, errorOnRequiredField } }`, &resp)

			require.EqualError(t, err, `[{"message":"boom","path":["errorBubble","errorOnRequiredField"]}]`)
			require.Nil(t, resp.ErrorBubble)
			require.Equal(t, "Ok", resp.Valid)
		})

		t.Run("when user returns null on required field", func(t *testing.T) {
			var resp struct {
				Valid       string
				ErrorBubble *struct {
					NilOnRequiredField string
				}
			}
			err := c.Post(`query { valid, errorBubble { id, nilOnRequiredField } }`, &resp)

			require.EqualError(t, err, `[{"message":"must not be null","path":["errorBubble","nilOnRequiredField"]}]`)
			require.Nil(t, resp.ErrorBubble)
			require.Equal(t, "Ok", resp.Valid)
		})
	})
}

type testResolver struct{}

func (r *testResolver) ForcedResolver() ForcedResolverResolver {
	return &forcedResolverResolver{nil}
}
func (r *testResolver) Query() QueryResolver {
	return &testQueryResolver{}
}

type testQueryResolver struct{ queryResolver }

func (r *testQueryResolver) ErrorBubble(ctx context.Context) (*Error, error) {
	return &Error{ID: "E1234"}, nil
}

func (r *testQueryResolver) Valid(ctx context.Context) (string, error) {
	return "Ok", nil
}

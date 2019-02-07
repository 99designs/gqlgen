package testserver

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestNullBubbling(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.Valid = func(ctx context.Context) (s string, e error) {
		return "Ok", nil
	}

	resolvers.QueryResolver.ErrorBubble = func(ctx context.Context) (i *Error, e error) {
		return &Error{ID: "E1234"}, nil
	}

	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{Resolvers: resolvers})))
	c := client.New(srv.URL)

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

	t.Run("null args", func(t *testing.T) {
		var resp struct {
			NullableArg *string
		}
		resolvers.QueryResolver.NullableArg = func(ctx context.Context, arg *int) (i *string, e error) {
			v := "Ok"
			return &v, nil
		}

		err := c.Post(`query { nullableArg(arg: null) }`, &resp)
		require.Nil(t, err)
		require.Equal(t, "Ok", *resp.NullableArg)
	})
}

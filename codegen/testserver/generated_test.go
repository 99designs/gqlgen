//go:generate rm -f resolver.go
//go:generate gorunpkg github.com/99designs/gqlgen

package testserver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"

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
	tickChan := make(chan string, 1)
	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{
		Resolvers: &testResolver{tick: tickChan},
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

	t.Run("subscriptions", func(t *testing.T) {
		t.Run("wont leak goroutines", func(t *testing.T) {
			initialGoroutineCount := runtime.NumGoroutine()

			sub := c.Websocket(`subscription { updated }`)

			tickChan <- "message"

			var msg struct {
				resp struct {
					Updated string
				}
			}

			err := sub.Next(&msg.resp)
			require.NoError(t, err)
			require.Equal(t, "message", msg.resp.Updated)
			sub.Close()

			// need a little bit of time for goroutines to settle
			time.Sleep(200 * time.Millisecond)

			require.Equal(t, initialGoroutineCount, runtime.NumGoroutine())
		})

		t.Run("will parse init payload", func(t *testing.T) {
			sub := c.WebsocketWithPayload(`subscription { initPayload }`, map[string]interface{}{
				"Authorization": "Bearer of the curse",
				"number":        32,
				"strings":       []string{"hello", "world"},
			})

			var msg struct {
				resp struct {
					InitPayload string
				}
			}

			err := sub.Next(&msg.resp)
			require.NoError(t, err)
			require.Equal(t, "AUTH:Bearer of the curse", msg.resp.InitPayload)
			err = sub.Next(&msg.resp)
			require.NoError(t, err)
			require.Equal(t, "Authorization = \"Bearer of the curse\"", msg.resp.InitPayload)
			err = sub.Next(&msg.resp)
			require.NoError(t, err)
			require.Equal(t, "number = 32", msg.resp.InitPayload)
			err = sub.Next(&msg.resp)
			require.NoError(t, err)
			require.Equal(t, "strings = []interface {}{\"hello\", \"world\"}", msg.resp.InitPayload)
			sub.Close()
		})
	})

	t.Run("custom directive implementation", func(t *testing.T) {
		t.Run("executes", func(t *testing.T) {
			var resp struct {
				DirectiveReturn string
			}
			c.MustPost(`query { directiveReturn }`, &resp)
			require.Equal(t, "CustomDirective", resp.DirectiveReturn)
		})
	})
}

func TestResponseExtension(t *testing.T) {
	srv := httptest.NewServer(handler.GraphQL(
		NewExecutableSchema(Config{
			Resolvers: &testResolver{},
		}),
		handler.RequestMiddleware(func(ctx context.Context, next func(ctx context.Context) []byte) []byte {
			rctx := graphql.GetRequestContext(ctx)
			if err := rctx.RegisterExtension("example", "value"); err != nil {
				panic(err)
			}
			return next(ctx)
		}),
	))
	c := client.New(srv.URL)

	raw, _ := c.RawPost(`query { valid }`)
	require.Equal(t, raw.Extensions["example"], "value")
}

type testResolver struct {
	tick chan string
}

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

func (r *testResolver) Subscription() SubscriptionResolver {
	return &testSubscriptionResolver{r}
}

type testSubscriptionResolver struct{ *testResolver }

func (r *testSubscriptionResolver) Updated(ctx context.Context) (<-chan string, error) {
	res := make(chan string, 1)

	go func() {
		for {
			select {
			case t := <-r.tick:
				res <- t
			case <-ctx.Done():
				close(res)
				return
			}
		}
	}()
	return res, nil
}

func (r *testSubscriptionResolver) InitPayload(ctx context.Context) (<-chan string, error) {
	payload := handler.GetInitPayload(ctx)
	channel := make(chan string, len(payload)+1)

	go func() {
		<-ctx.Done()
		close(channel)
	}()

	// Test the helper function separately
	auth := payload.Authorization()
	if auth != "" {
		channel <- "AUTH:" + auth
	} else {
		channel <- "AUTH:NONE"
	}

	// Send them over the channel in alphabetic order
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		channel <- fmt.Sprintf("%s = %#+v", key, payload[key])
	}

	return channel, nil
}

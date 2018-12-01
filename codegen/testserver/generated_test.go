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
	"sync"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/introspection"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/assert"
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
	resolvers := &testResolver{tick: make(chan string, 1)}

	srv := httptest.NewServer(
		handler.GraphQL(
			NewExecutableSchema(Config{Resolvers: resolvers}),
			handler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
				path, _ := ctx.Value("path").([]int)
				return next(context.WithValue(ctx, "path", append(path, 1)))
			}),
			handler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
				path, _ := ctx.Value("path").([]int)
				return next(context.WithValue(ctx, "path", append(path, 2)))
			}),
		))
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

	t.Run("middleware", func(t *testing.T) {
		var resp struct {
			User struct {
				ID      int
				Friends []struct {
					ID int
				}
			}
		}

		called := false
		resolvers.userFriends = func(ctx context.Context, obj *User) ([]User, error) {
			assert.Equal(t, []int{1, 2, 1, 2}, ctx.Value("path"))
			called = true
			return []User{}, nil
		}

		err := c.Post(`query { user(id: 1) { id, friends { id } } }`, &resp)

		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("subscriptions", func(t *testing.T) {
		t.Run("wont leak goroutines", func(t *testing.T) {
			initialGoroutineCount := runtime.NumGoroutine()

			sub := c.Websocket(`subscription { updated }`)

			resolvers.tick <- "message"

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
			start := time.Now()
			for time.Since(start).Seconds() < 2 && initialGoroutineCount != runtime.NumGoroutine() {
				time.Sleep(5 * time.Millisecond)
			}

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

	t.Run("null args", func(t *testing.T) {
		var resp struct {
			NullableArg *string
		}
		err := c.Post(`query { nullableArg(arg: null) }`, &resp)
		require.Nil(t, err)
		require.Equal(t, "Ok", *resp.NullableArg)
	})
}

func TestIntrospection(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		resolvers := &testResolver{tick: make(chan string, 1)}

		srv := httptest.NewServer(
			handler.GraphQL(
				NewExecutableSchema(Config{Resolvers: resolvers}),
				handler.IntrospectionEnabled(false),
			),
		)

		c := client.New(srv.URL)

		var resp interface{}
		err := c.Post(introspection.Query, &resp)
		require.EqualError(t, err, "[{\"message\":\"introspection disabled\",\"path\":[\"__schema\"]}]")
	})

	t.Run("enabled by default", func(t *testing.T) {
		resolvers := &testResolver{tick: make(chan string, 1)}

		srv := httptest.NewServer(
			handler.GraphQL(
				NewExecutableSchema(Config{Resolvers: resolvers}),
			),
		)

		c := client.New(srv.URL)

		var resp interface{}
		err := c.Post(introspection.Query, &resp)
		require.NoError(t, err)

		t.Run("does not return empty deprecation strings", func(t *testing.T) {
			q := `{
			  __type(name:"InnerObject") {
			    fields {
			      name
			      deprecationReason
			    }
			  }
			}`

			c := client.New(srv.URL)
			var resp struct {
				Type struct {
					Fields []struct {
						Name              string
						DeprecationReason *string
					}
				} `json:"__type"`
			}
			err := c.Post(q, &resp)
			require.NoError(t, err)

			require.Equal(t, "id", resp.Type.Fields[0].Name)
			require.Nil(t, resp.Type.Fields[0].DeprecationReason)
		})
	})

	t.Run("disabled by middleware", func(t *testing.T) {
		resolvers := &testResolver{tick: make(chan string, 1)}

		srv := httptest.NewServer(
			handler.GraphQL(
				NewExecutableSchema(Config{Resolvers: resolvers}),
				handler.RequestMiddleware(func(ctx context.Context, next func(ctx context.Context) []byte) []byte {
					graphql.GetRequestContext(ctx).DisableIntrospection = true

					return next(ctx)
				}),
			),
		)

		c := client.New(srv.URL)

		var resp interface{}
		err := c.Post(introspection.Query, &resp)
		require.EqualError(t, err, "[{\"message\":\"introspection disabled\",\"path\":[\"__schema\"]}]")
	})

}

var _ graphql.Tracer = (*testTracer)(nil)

type testTracer struct {
	id     int
	append func(string)
}

func (tt *testTracer) StartOperationParsing(ctx context.Context) context.Context {
	line := fmt.Sprintf("op:p:start:%d", tt.id)

	tracerLogs, _ := ctx.Value("tracer").([]string)
	ctx = context.WithValue(ctx, "tracer", append(append([]string{}, tracerLogs...), line))
	tt.append(line)

	return ctx
}

func (tt *testTracer) EndOperationParsing(ctx context.Context) {
	tt.append(fmt.Sprintf("op:p:end:%d", tt.id))
}

func (tt *testTracer) StartOperationValidation(ctx context.Context) context.Context {
	line := fmt.Sprintf("op:v:start:%d", tt.id)

	tracerLogs, _ := ctx.Value("tracer").([]string)
	ctx = context.WithValue(ctx, "tracer", append(append([]string{}, tracerLogs...), line))
	tt.append(line)

	return ctx
}

func (tt *testTracer) EndOperationValidation(ctx context.Context) {
	tt.append(fmt.Sprintf("op:v:end:%d", tt.id))
}

func (tt *testTracer) StartOperationExecution(ctx context.Context) context.Context {
	line := fmt.Sprintf("op:e:start:%d", tt.id)

	tracerLogs, _ := ctx.Value("tracer").([]string)
	ctx = context.WithValue(ctx, "tracer", append(append([]string{}, tracerLogs...), line))
	tt.append(line)

	return ctx
}

func (tt *testTracer) StartFieldExecution(ctx context.Context, field graphql.CollectedField) context.Context {
	line := fmt.Sprintf("field'a:e:start:%d:%s", tt.id, field.Name)

	tracerLogs, _ := ctx.Value("tracer").([]string)
	ctx = context.WithValue(ctx, "tracer", append(append([]string{}, tracerLogs...), line))
	tt.append(line)

	return ctx
}

func (tt *testTracer) StartFieldResolverExecution(ctx context.Context, rc *graphql.ResolverContext) context.Context {
	line := fmt.Sprintf("field'b:e:start:%d:%v", tt.id, rc.Path())

	tracerLogs, _ := ctx.Value("tracer").([]string)
	ctx = context.WithValue(ctx, "tracer", append(append([]string{}, tracerLogs...), line))
	tt.append(line)

	return ctx
}

func (tt *testTracer) StartFieldChildExecution(ctx context.Context) context.Context {
	line := fmt.Sprintf("field'c:e:start:%d", tt.id)

	tracerLogs, _ := ctx.Value("tracer").([]string)
	ctx = context.WithValue(ctx, "tracer", append(append([]string{}, tracerLogs...), line))
	tt.append(line)

	return ctx
}

func (tt *testTracer) EndFieldExecution(ctx context.Context) {
	tt.append(fmt.Sprintf("field:e:end:%d", tt.id))
}

func (tt *testTracer) EndOperationExecution(ctx context.Context) {
	tt.append(fmt.Sprintf("op:e:end:%d", tt.id))
}

var _ graphql.Tracer = (*configurableTracer)(nil)

type configurableTracer struct {
	StartOperationParsingCallback       func(ctx context.Context) context.Context
	EndOperationParsingCallback         func(ctx context.Context)
	StartOperationValidationCallback    func(ctx context.Context) context.Context
	EndOperationValidationCallback      func(ctx context.Context)
	StartOperationExecutionCallback     func(ctx context.Context) context.Context
	StartFieldExecutionCallback         func(ctx context.Context, field graphql.CollectedField) context.Context
	StartFieldResolverExecutionCallback func(ctx context.Context, rc *graphql.ResolverContext) context.Context
	StartFieldChildExecutionCallback    func(ctx context.Context) context.Context
	EndFieldExecutionCallback           func(ctx context.Context)
	EndOperationExecutionCallback       func(ctx context.Context)
}

func (ct *configurableTracer) StartOperationParsing(ctx context.Context) context.Context {
	if f := ct.StartOperationParsingCallback; f != nil {
		ctx = f(ctx)
	}
	return ctx
}

func (ct *configurableTracer) EndOperationParsing(ctx context.Context) {
	if f := ct.EndOperationParsingCallback; f != nil {
		f(ctx)
	}
}

func (ct *configurableTracer) StartOperationValidation(ctx context.Context) context.Context {
	if f := ct.StartOperationValidationCallback; f != nil {
		ctx = f(ctx)
	}
	return ctx
}

func (ct *configurableTracer) EndOperationValidation(ctx context.Context) {
	if f := ct.EndOperationValidationCallback; f != nil {
		f(ctx)
	}
}

func (ct *configurableTracer) StartOperationExecution(ctx context.Context) context.Context {
	if f := ct.StartOperationExecutionCallback; f != nil {
		ctx = f(ctx)
	}
	return ctx
}

func (ct *configurableTracer) StartFieldExecution(ctx context.Context, field graphql.CollectedField) context.Context {
	if f := ct.StartFieldExecutionCallback; f != nil {
		ctx = f(ctx, field)
	}
	return ctx
}

func (ct *configurableTracer) StartFieldResolverExecution(ctx context.Context, rc *graphql.ResolverContext) context.Context {
	if f := ct.StartFieldResolverExecutionCallback; f != nil {
		ctx = f(ctx, rc)
	}
	return ctx
}

func (ct *configurableTracer) StartFieldChildExecution(ctx context.Context) context.Context {
	if f := ct.StartFieldChildExecutionCallback; f != nil {
		ctx = f(ctx)
	}
	return ctx
}

func (ct *configurableTracer) EndFieldExecution(ctx context.Context) {
	if f := ct.EndFieldExecutionCallback; f != nil {
		f(ctx)
	}
}

func (ct *configurableTracer) EndOperationExecution(ctx context.Context) {
	if f := ct.EndOperationExecutionCallback; f != nil {
		f(ctx)
	}
}

func TestTracer(t *testing.T) {
	t.Run("called in the correct order", func(t *testing.T) {
		resolvers := &testResolver{tick: make(chan string, 1)}

		var tracerLog []string
		var mu sync.Mutex

		srv := httptest.NewServer(
			handler.GraphQL(
				NewExecutableSchema(Config{Resolvers: resolvers}),
				handler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
					path, _ := ctx.Value("path").([]int)
					return next(context.WithValue(ctx, "path", append(path, 1)))
				}),
				handler.ResolverMiddleware(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
					path, _ := ctx.Value("path").([]int)
					return next(context.WithValue(ctx, "path", append(path, 2)))
				}),
				handler.Tracer(&testTracer{
					id: 1,
					append: func(s string) {
						mu.Lock()
						defer mu.Unlock()
						tracerLog = append(tracerLog, s)
					},
				}),
				handler.Tracer(&testTracer{
					id: 2,
					append: func(s string) {
						mu.Lock()
						defer mu.Unlock()
						tracerLog = append(tracerLog, s)
					},
				}),
			))
		defer srv.Close()
		c := client.New(srv.URL)

		var resp struct {
			User struct {
				ID      int
				Friends []struct {
					ID int
				}
			}
		}

		called := false
		resolvers.userFriends = func(ctx context.Context, obj *User) ([]User, error) {
			assert.Equal(t, []string{
				"op:p:start:1", "op:p:start:2",
				"op:v:start:1", "op:v:start:2",
				"op:e:start:1", "op:e:start:2",
				"field'a:e:start:1:user", "field'a:e:start:2:user",
				"field'b:e:start:1:[user]", "field'b:e:start:2:[user]",
				"field'c:e:start:1", "field'c:e:start:2",
				"field'a:e:start:1:friends", "field'a:e:start:2:friends",
				"field'b:e:start:1:[user friends]", "field'b:e:start:2:[user friends]",
			}, ctx.Value("tracer"))
			called = true
			return []User{}, nil
		}

		err := c.Post(`query { user(id: 1) { id, friends { id } } }`, &resp)

		require.NoError(t, err)
		require.True(t, called)
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{
			"op:p:start:1", "op:p:start:2",
			"op:p:end:2", "op:p:end:1",

			"op:v:start:1", "op:v:start:2",
			"op:v:end:2", "op:v:end:1",

			"op:e:start:1", "op:e:start:2",

			"field'a:e:start:1:user", "field'a:e:start:2:user",
			"field'b:e:start:1:[user]", "field'b:e:start:2:[user]",
			"field'c:e:start:1", "field'c:e:start:2",
			"field'a:e:start:1:id", "field'a:e:start:2:id",
			"field'b:e:start:1:[user id]", "field'b:e:start:2:[user id]",
			"field'c:e:start:1", "field'c:e:start:2",
			"field:e:end:2", "field:e:end:1",
			"field'a:e:start:1:friends", "field'a:e:start:2:friends",
			"field'b:e:start:1:[user friends]", "field'b:e:start:2:[user friends]",
			"field'c:e:start:1", "field'c:e:start:2",
			"field:e:end:2", "field:e:end:1",
			"field:e:end:2", "field:e:end:1",

			"op:e:end:2", "op:e:end:1",
		}, tracerLog)
	})

	t.Run("take ctx over from prev step", func(t *testing.T) {
		resolvers := &testResolver{tick: make(chan string, 1)}

		configurableTracer := &configurableTracer{
			StartOperationParsingCallback: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "StartOperationParsing", true)
			},
			EndOperationParsingCallback: func(ctx context.Context) {
				assert.NotNil(t, ctx.Value("StartOperationParsing"))
			},

			StartOperationValidationCallback: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "StartOperationValidation", true)
			},
			EndOperationValidationCallback: func(ctx context.Context) {
				assert.NotNil(t, ctx.Value("StartOperationParsing"))
				assert.NotNil(t, ctx.Value("StartOperationValidation"))
			},

			StartOperationExecutionCallback: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "StartOperationExecution", true)
			},
			StartFieldExecutionCallback: func(ctx context.Context, field graphql.CollectedField) context.Context {
				return context.WithValue(ctx, "StartFieldExecution", true)
			},
			StartFieldResolverExecutionCallback: func(ctx context.Context, rc *graphql.ResolverContext) context.Context {
				return context.WithValue(ctx, "StartFieldResolverExecution", true)
			},
			StartFieldChildExecutionCallback: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, "StartFieldChildExecution", true)
			},
			EndFieldExecutionCallback: func(ctx context.Context) {
				assert.NotNil(t, ctx.Value("StartOperationParsing"))
				assert.NotNil(t, ctx.Value("StartOperationValidation"))
				assert.NotNil(t, ctx.Value("StartOperationExecution"))
				assert.NotNil(t, ctx.Value("StartFieldExecution"))
				assert.NotNil(t, ctx.Value("StartFieldResolverExecution"))
				assert.NotNil(t, ctx.Value("StartFieldChildExecution"))
			},

			EndOperationExecutionCallback: func(ctx context.Context) {
				assert.NotNil(t, ctx.Value("StartOperationParsing"))
				assert.NotNil(t, ctx.Value("StartOperationValidation"))
				assert.NotNil(t, ctx.Value("StartOperationExecution"))
			},
		}

		srv := httptest.NewServer(
			handler.GraphQL(
				NewExecutableSchema(Config{Resolvers: resolvers}),
				handler.Tracer(configurableTracer),
			))
		defer srv.Close()
		c := client.New(srv.URL)

		var resp struct {
			User struct {
				ID      int
				Friends []struct {
					ID int
				}
			}
		}

		called := false
		resolvers.userFriends = func(ctx context.Context, obj *User) ([]User, error) {
			called = true
			return []User{}, nil
		}

		err := c.Post(`query { user(id: 1) { id, friends { id } } }`, &resp)

		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("model methods", func(t *testing.T) {
		srv := httptest.NewServer(
			handler.GraphQL(
				NewExecutableSchema(Config{Resolvers: &testResolver{}}),
			))
		defer srv.Close()
		c := client.New(srv.URL)
		t.Run("without context", func(t *testing.T) {
			var resp struct {
				ModelMethods struct {
					NoContext bool
				}
			}
			err := c.Post(`query { modelMethods{ noContext } }`, &resp)
			require.NoError(t, err)
			require.True(t, resp.ModelMethods.NoContext)
		})
		t.Run("with context", func(t *testing.T) {
			var resp struct {
				ModelMethods struct {
					WithContext bool
				}
			}
			err := c.Post(`query { modelMethods{ withContext } }`, &resp)
			require.NoError(t, err)
			require.True(t, resp.ModelMethods.WithContext)
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
	tick        chan string
	userFriends func(ctx context.Context, obj *User) ([]User, error)
}

func (r *testResolver) ForcedResolver() ForcedResolverResolver {
	return &forcedResolverResolver{nil}
}

func (r *testResolver) User() UserResolver {
	return &testUserResolver{r}
}

func (r *testResolver) Query() QueryResolver {
	return &testQueryResolver{}
}
func (r *testResolver) ModelMethods() ModelMethodsResolver {
	return &testModelMethodsResolver{}
}

type testModelMethodsResolver struct{}

func (r *testModelMethodsResolver) ResolverField(ctx context.Context, obj *ModelMethods) (bool, error) {
	return true, nil
}

type testQueryResolver struct{ queryResolver }

func (r *testQueryResolver) ErrorBubble(ctx context.Context) (*Error, error) {
	return &Error{ID: "E1234"}, nil
}

func (r *testQueryResolver) Valid(ctx context.Context) (string, error) {
	return "Ok", nil
}

func (r *testQueryResolver) User(ctx context.Context, id int) (User, error) {
	return User{ID: 1}, nil
}

func (r *testQueryResolver) NullableArg(ctx context.Context, arg *int) (*string, error) {
	s := "Ok"
	return &s, nil
}

func (r *testQueryResolver) ModelMethods(ctx context.Context) (*ModelMethods, error) {
	return &ModelMethods{}, nil
}

func (r *testResolver) Subscription() SubscriptionResolver {
	return &testSubscriptionResolver{r}
}

type testUserResolver struct{ *testResolver }

func (r *testResolver) Friends(ctx context.Context, obj *User) ([]User, error) {
	return r.userFriends(ctx, obj)
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

package testserver

import (
	"context"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracer(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.User = func(ctx context.Context, id int) (user *User, e error) {
		return &User{ID: 1}, nil
	}
	t.Run("called in the correct order", func(t *testing.T) {
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
		resolvers.UserResolver.Friends = func(ctx context.Context, obj *User) ([]User, error) {
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
		resolvers.UserResolver.Friends = func(ctx context.Context, obj *User) ([]User, error) {
			called = true
			return []User{}, nil
		}

		err := c.Post(`query { user(id: 1) { id, friends { id } } }`, &resp)

		require.NoError(t, err)
		require.True(t, called)
	})
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

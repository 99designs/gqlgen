package testserver

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/handler"
	"github.com/stretchr/testify/require"
)

func TestDirectives(t *testing.T) {
	resolvers := &Stub{}
	resolvers.QueryResolver.DirectiveArg = func(ctx context.Context, arg string) (i *string, e error) {
		s := "Ok"
		return &s, nil
	}

	resolvers.QueryResolver.DirectiveInput = func(ctx context.Context, arg InputDirectives) (i *string, e error) {
		s := "Ok"
		return &s, nil
	}

	resolvers.QueryResolver.DirectiveInputNullable = func(ctx context.Context, arg *InputDirectives) (i *string, e error) {
		s := "Ok"
		return &s, nil
	}

	resolvers.QueryResolver.DirectiveNullableArg = func(ctx context.Context, arg *int, arg2 *int) (i *string, e error) {
		s := "Ok"
		return &s, nil
	}

	resolvers.QueryResolver.DirectiveInputType = func(ctx context.Context, arg InnerInput) (i *string, e error) {
		s := "Ok"
		return &s, nil
	}

	srv := httptest.NewServer(
		handler.GraphQL(
			NewExecutableSchema(Config{
				Resolvers: resolvers,
				Directives: DirectiveRoot{
					Length: func(ctx context.Context, obj interface{}, next graphql.Resolver, min int, max *int) (interface{}, error) {
						res, err := next(ctx)
						if err != nil {
							return nil, err
						}

						s := res.(string)
						if len(s) < min {
							return nil, fmt.Errorf("too short")
						}
						if max != nil && len(s) > *max {
							return nil, fmt.Errorf("too long")
						}
						return res, nil
					},
					Range: func(ctx context.Context, obj interface{}, next graphql.Resolver, min *int, max *int) (interface{}, error) {
						res, err := next(ctx)
						if err != nil {
							return nil, err
						}

						switch res := res.(type) {
						case int:
							if min != nil && res < *min {
								return nil, fmt.Errorf("too small")
							}
							if max != nil && res > *max {
								return nil, fmt.Errorf("too large")
							}
							return next(ctx)

						case int64:
							if min != nil && int(res) < *min {
								return nil, fmt.Errorf("too small")
							}
							if max != nil && int(res) > *max {
								return nil, fmt.Errorf("too large")
							}
							return next(ctx)

						case *int:
							if min != nil && *res < *min {
								return nil, fmt.Errorf("too small")
							}
							if max != nil && *res > *max {
								return nil, fmt.Errorf("too large")
							}
							return next(ctx)
						}
						return nil, fmt.Errorf("unsupported type %T", res)
					},
					Custom: func(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
						return next(ctx)
					},
				},
			}),
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

	t.Run("arg directives", func(t *testing.T) {
		t.Run("when function errors on directives", func(t *testing.T) {
			var resp struct {
				DirectiveArg *string
			}

			err := c.Post(`query { directiveArg(arg: "") }`, &resp)

			require.EqualError(t, err, `[{"message":"too short","path":["directiveArg"]}]`)
			require.Nil(t, resp.DirectiveArg)
		})
		t.Run("when function errors on nullable arg directives", func(t *testing.T) {
			var resp struct {
				DirectiveNullableArg *string
			}

			err := c.Post(`query { directiveNullableArg(arg: -100) }`, &resp)

			require.EqualError(t, err, `[{"message":"too small","path":["directiveNullableArg"]}]`)
			require.Nil(t, resp.DirectiveNullableArg)
		})
		t.Run("when function success on nullable arg directives", func(t *testing.T) {
			var resp struct {
				DirectiveNullableArg *string
			}

			err := c.Post(`query { directiveNullableArg }`, &resp)

			require.Nil(t, err)
			require.Equal(t, "Ok", *resp.DirectiveNullableArg)
		})
		t.Run("when function success on valid nullable arg directives", func(t *testing.T) {
			var resp struct {
				DirectiveNullableArg *string
			}

			err := c.Post(`query { directiveNullableArg(arg: 1) }`, &resp)

			require.Nil(t, err)
			require.Equal(t, "Ok", *resp.DirectiveNullableArg)
		})
		t.Run("when function success", func(t *testing.T) {
			var resp struct {
				DirectiveArg *string
			}

			err := c.Post(`query { directiveArg(arg: "test") }`, &resp)

			require.Nil(t, err)
			require.Equal(t, "Ok", *resp.DirectiveArg)
		})
	})
	t.Run("input field directives", func(t *testing.T) {
		t.Run("when function errors on directives", func(t *testing.T) {
			var resp struct {
				DirectiveInputNullable *string
			}

			err := c.Post(`query { directiveInputNullable(arg: {text:"invalid text",inner:{message:"123"}}) }`, &resp)

			require.EqualError(t, err, `[{"message":"too long","path":["directiveInputNullable"]}]`)
			require.Nil(t, resp.DirectiveInputNullable)
		})
		t.Run("when function errors on inner directives", func(t *testing.T) {
			var resp struct {
				DirectiveInputNullable *string
			}

			err := c.Post(`query { directiveInputNullable(arg: {text:"2",inner:{message:""}}) }`, &resp)

			require.EqualError(t, err, `[{"message":"too short","path":["directiveInputNullable"]}]`)
			require.Nil(t, resp.DirectiveInputNullable)
		})
		t.Run("when function errors on nullable inner directives", func(t *testing.T) {
			var resp struct {
				DirectiveInputNullable *string
			}

			err := c.Post(`query { directiveInputNullable(arg: {text:"success",inner:{message:"1"},innerNullable:{message:""}}) }`, &resp)

			require.EqualError(t, err, `[{"message":"too short","path":["directiveInputNullable"]}]`)
			require.Nil(t, resp.DirectiveInputNullable)
		})
		t.Run("when function success", func(t *testing.T) {
			var resp struct {
				DirectiveInputNullable *string
			}

			err := c.Post(`query { directiveInputNullable(arg: {text:"23",inner:{message:"1"}}) }`, &resp)

			require.Nil(t, err)
			require.Equal(t, "Ok", *resp.DirectiveInputNullable)
		})
		t.Run("when function inner nullable success", func(t *testing.T) {
			var resp struct {
				DirectiveInputNullable *string
			}

			err := c.Post(`query { directiveInputNullable(arg: {text:"23",inner:{message:"1"},innerNullable:{message:"success"}}) }`, &resp)

			require.Nil(t, err)
			require.Equal(t, "Ok", *resp.DirectiveInputNullable)
		})
		t.Run("when arg has directive", func(t *testing.T) {
			var resp struct {
				DirectiveInputType *string
			}

			err := c.Post(`query { directiveInputType(arg: {id: 1}) }`, &resp)

			require.Nil(t, err)
			require.Equal(t, "Ok", *resp.DirectiveInputType)
		})
	})
}

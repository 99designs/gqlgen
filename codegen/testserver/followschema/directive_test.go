package followschema

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
)

// isNil checks if the given value is nil
func isNil(input any) bool {
	if input == nil {
		return true
	}
	// Using reflect to check if the value is nil. This is necessary for
	// any types that are not nil types but have a nil value (e.g. *string).
	value := reflect.ValueOf(input)
	return value.IsNil()
}

type ckey string

func TestDirectives(t *testing.T) {
	resolvers := &Stub{}
	ok := "Ok"
	resolvers.QueryResolver.DirectiveArg = func(ctx context.Context, arg string) (i *string, e error) {
		return &ok, nil
	}

	resolvers.QueryResolver.DirectiveInput = func(ctx context.Context, arg InputDirectives) (i *string, e error) {
		return &ok, nil
	}

	resolvers.QueryResolver.DirectiveInputNullable = func(ctx context.Context, arg *InputDirectives) (i *string, e error) {
		return &ok, nil
	}

	resolvers.QueryResolver.DirectiveNullableArg = func(ctx context.Context, arg *int, arg2 *int, arg3 *string) (*string, error) {
		return &ok, nil
	}

	resolvers.QueryResolver.DirectiveSingleNullableArg = func(ctx context.Context, arg1 *string) (*string, error) {
		if arg1 != nil {
			return arg1, nil
		}

		return &ok, nil
	}

	resolvers.QueryResolver.DirectiveInputType = func(ctx context.Context, arg InnerInput) (i *string, e error) {
		return &ok, nil
	}

	resolvers.QueryResolver.DirectiveObject = func(ctx context.Context) (*ObjectDirectives, error) {
		return &ObjectDirectives{
			Text:         ok,
			NullableText: &ok,
		}, nil
	}

	resolvers.QueryResolver.DirectiveObjectWithCustomGoModel = func(ctx context.Context) (*ObjectDirectivesWithCustomGoModel, error) {
		return &ObjectDirectivesWithCustomGoModel{
			NullableText: ok,
		}, nil
	}

	resolvers.QueryResolver.DirectiveField = func(ctx context.Context) (*string, error) {
		if s, ok := ctx.Value(ckey("request_id")).(*string); ok {
			return s, nil
		}

		return nil, nil
	}

	resolvers.QueryResolver.DirectiveDouble = func(ctx context.Context) (*string, error) {
		return &ok, nil
	}

	resolvers.QueryResolver.DirectiveUnimplemented = func(ctx context.Context) (*string, error) {
		return &ok, nil
	}

	okchan := func() (<-chan *string, error) {
		res := make(chan *string, 1)
		res <- &ok
		close(res)
		return res, nil
	}

	resolvers.SubscriptionResolver.DirectiveArg = func(ctx context.Context, arg string) (strings <-chan *string, e error) {
		return okchan()
	}

	resolvers.SubscriptionResolver.DirectiveNullableArg = func(ctx context.Context, arg *int, arg2 *int, arg3 *string) (strings <-chan *string, e error) {
		return okchan()
	}

	resolvers.SubscriptionResolver.DirectiveDouble = func(ctx context.Context) (strings <-chan *string, e error) {
		return okchan()
	}

	resolvers.SubscriptionResolver.DirectiveUnimplemented = func(ctx context.Context) (<-chan *string, error) {
		return okchan()
	}
	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			//nolint:revive // can't rename min, max because it's generated code
			Length: func(ctx context.Context, obj any, next graphql.Resolver, min int, max *int, message *string) (any, error) {
				e := func(msg string) error {
					if message == nil {
						return errors.New(msg)
					}
					return errors.New(*message)
				}
				res, err := next(ctx)
				if err != nil {
					return nil, err
				}

				s := res.(string)
				if len(s) < min {
					return nil, e("too short")
				}
				if max != nil && len(s) > *max {
					return nil, e("too long")
				}
				return res, nil
			},
			//nolint:revive // can't rename min, max because it's generated code
			Range: func(ctx context.Context, obj any, next graphql.Resolver, min *int, max *int) (any, error) {
				res, err := next(ctx)
				if err != nil {
					return nil, err
				}

				switch res := res.(type) {
				case int:
					if min != nil && res < *min {
						return nil, errors.New("too small")
					}
					if max != nil && res > *max {
						return nil, errors.New("too large")
					}
					return next(ctx)

				case int64:
					if min != nil && int(res) < *min {
						return nil, errors.New("too small")
					}
					if max != nil && int(res) > *max {
						return nil, errors.New("too large")
					}
					return next(ctx)

				case *int:
					if min != nil && *res < *min {
						return nil, errors.New("too small")
					}
					if max != nil && *res > *max {
						return nil, errors.New("too large")
					}
					return next(ctx)
				}
				return nil, fmt.Errorf("unsupported type %T", res)
			},
			Custom: func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
				return next(ctx)
			},
			Logged: func(ctx context.Context, obj any, next graphql.Resolver, id string) (any, error) {
				return next(context.WithValue(ctx, ckey("request_id"), &id))
			},
			ToNull: func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
				return nil, nil
			},
			Populate: func(ctx context.Context, obj any, next graphql.Resolver, value string) (any, error) {
				res, err := next(ctx)
				if err != nil {
					return nil, err
				}

				if !isNil(res) {
					return res, err
				}

				return &value, nil
			},
			Noop: func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
				return next(ctx)
			},
			Directive1: func(ctx context.Context, obj any, next graphql.Resolver) (res any, err error) {
				return next(ctx)
			},
			Directive2: func(ctx context.Context, obj any, next graphql.Resolver) (res any, err error) {
				return next(ctx)
			},
			Directive3: func(ctx context.Context, obj any, next graphql.Resolver) (res any, err error) {
				return next(ctx)
			},
			Order1: func(ctx context.Context, obj any, next graphql.Resolver, location string) (res any, err error) {
				order := []string{location}
				res, err = next(ctx)
				od := res.(*ObjectDirectives)
				od.Order = append(order, od.Order...)
				return od, err
			},
			Order2: func(ctx context.Context, obj any, next graphql.Resolver, location string) (res any, err error) {
				order := []string{location}
				res, err = next(ctx)
				od := res.(*ObjectDirectives)
				od.Order = append(order, od.Order...)
				return od, err
			},
			Unimplemented: nil,
		},
	}))
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: time.Second,
	})
	srv.AddTransport(transport.POST{})
	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res any, err error) {
		path, _ := ctx.Value(ckey("path")).([]int)
		return next(context.WithValue(ctx, ckey("path"), append(path, 1)))
	})

	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res any, err error) {
		path, _ := ctx.Value(ckey("path")).([]int)
		return next(context.WithValue(ctx, ckey("path"), append(path, 2)))
	})

	c := client.New(srv)

	t.Run("arg directives", func(t *testing.T) {
		t.Run("when function errors on directives", func(t *testing.T) {
			var resp struct {
				DirectiveArg *string
			}

			err := c.Post(`query { directiveArg(arg: "") }`, &resp)

			require.EqualError(t, err, `[{"message":"invalid length","path":["directiveArg","arg"]}]`)
			require.Nil(t, resp.DirectiveArg)
		})
		t.Run("when function errors on nullable arg directives", func(t *testing.T) {
			var resp struct {
				DirectiveNullableArg *string
			}

			err := c.Post(`query { directiveNullableArg(arg: -100) }`, &resp)

			require.EqualError(t, err, `[{"message":"too small","path":["directiveNullableArg","arg"]}]`)
			require.Nil(t, resp.DirectiveNullableArg)
		})
		t.Run("when function success on nullable arg directives", func(t *testing.T) {
			var resp struct {
				DirectiveNullableArg *string
			}

			err := c.Post(`query { directiveNullableArg }`, &resp)

			require.NoError(t, err)
			require.Equal(t, "Ok", *resp.DirectiveNullableArg)
		})
		t.Run("when function success on valid nullable arg directives", func(t *testing.T) {
			var resp struct {
				DirectiveNullableArg *string
			}

			err := c.Post(`query { directiveNullableArg(arg: 1) }`, &resp)

			require.NoError(t, err)
			require.Equal(t, "Ok", *resp.DirectiveNullableArg)
		})
		t.Run("when function success", func(t *testing.T) {
			var resp struct {
				DirectiveArg *string
			}

			err := c.Post(`query { directiveArg(arg: "test") }`, &resp)

			require.NoError(t, err)
			require.Equal(t, "Ok", *resp.DirectiveArg)
		})

		t.Run("directive is not called with null arguments", func(t *testing.T) {
			var resp struct {
				DirectiveSingleNullableArg *string
			}

			err := c.Post(`query { directiveSingleNullableArg }`, &resp)

			require.NoError(t, err)
			require.Equal(t, "Ok", *resp.DirectiveSingleNullableArg)
		})
	})
	t.Run("field definition directives", func(t *testing.T) {
		resolvers.QueryResolver.DirectiveFieldDef = func(ctx context.Context, ret string) (i string, e error) {
			return ret, nil
		}

		t.Run("too short", func(t *testing.T) {
			var resp struct {
				DirectiveFieldDef string
			}

			err := c.Post(`query { directiveFieldDef(ret: "") }`, &resp)

			require.EqualError(t, err, `[{"message":"not valid","path":["directiveFieldDef"]}]`)
		})

		t.Run("has 2 directives", func(t *testing.T) {
			var resp struct {
				DirectiveDouble string
			}

			c.MustPost(`query { directiveDouble }`, &resp)

			require.Equal(t, "Ok", resp.DirectiveDouble)
		})

		t.Run("directive is not implemented", func(t *testing.T) {
			var resp struct {
				DirectiveUnimplemented string
			}

			err := c.Post(`query { directiveUnimplemented }`, &resp)

			require.EqualError(t, err, `[{"message":"directive unimplemented is not implemented","path":["directiveUnimplemented"]}]`)
		})

		t.Run("ok", func(t *testing.T) {
			var resp struct {
				DirectiveFieldDef string
			}

			c.MustPost(`query { directiveFieldDef(ret: "aaa") }`, &resp)

			require.Equal(t, "aaa", resp.DirectiveFieldDef)
		})
	})
	t.Run("field directives", func(t *testing.T) {
		t.Run("add field directive", func(t *testing.T) {
			var resp struct {
				DirectiveField string
			}

			c.MustPost(`query { directiveField@logged(id:"testes_id") }`, &resp)

			require.Equal(t, `testes_id`, resp.DirectiveField)
		})
		t.Run("without field directive", func(t *testing.T) {
			var resp struct {
				DirectiveField *string
			}

			c.MustPost(`query { directiveField }`, &resp)

			require.Nil(t, resp.DirectiveField)
		})
	})
	t.Run("input field directives", func(t *testing.T) {
		t.Run("when function errors on directives", func(t *testing.T) {
			var resp struct {
				DirectiveInputNullable *string
			}

			err := c.Post(`query { directiveInputNullable(arg: {text:"invalid text",inner:{message:"123"}}) }`, &resp)

			require.EqualError(t, err, `[{"message":"not valid","path":["directiveInputNullable","arg","text"]}]`)
			require.Nil(t, resp.DirectiveInputNullable)
		})
		t.Run("when function errors on inner directives", func(t *testing.T) {
			var resp struct {
				DirectiveInputNullable *string
			}

			err := c.Post(`query { directiveInputNullable(arg: {text:"2",inner:{message:""}}) }`, &resp)

			require.EqualError(t, err, `[{"message":"not valid","path":["directiveInputNullable","arg","inner","message"]}]`)
			require.Nil(t, resp.DirectiveInputNullable)
		})
		t.Run("when function errors on nullable inner directives", func(t *testing.T) {
			var resp struct {
				DirectiveInputNullable *string
			}

			err := c.Post(`query { directiveInputNullable(arg: {text:"success",inner:{message:"1"},innerNullable:{message:""}}) }`, &resp)

			require.EqualError(t, err, `[{"message":"not valid","path":["directiveInputNullable","arg","innerNullable","message"]}]`)
			require.Nil(t, resp.DirectiveInputNullable)
		})
		t.Run("when function success", func(t *testing.T) {
			var resp struct {
				DirectiveInputNullable *string
			}

			err := c.Post(`query { directiveInputNullable(arg: {text:"23",inner:{message:"1"}}) }`, &resp)

			require.NoError(t, err)
			require.Equal(t, "Ok", *resp.DirectiveInputNullable)
		})
		t.Run("when function inner nullable success", func(t *testing.T) {
			var resp struct {
				DirectiveInputNullable *string
			}

			err := c.Post(`query { directiveInputNullable(arg: {text:"23",nullableText:"23",inner:{message:"1"},innerNullable:{message:"success"}}) }`, &resp)

			require.NoError(t, err)
			require.Equal(t, "Ok", *resp.DirectiveInputNullable)
		})
		t.Run("when arg has directive", func(t *testing.T) {
			var resp struct {
				DirectiveInputType *string
			}

			err := c.Post(`query { directiveInputType(arg: {id: 1}) }`, &resp)

			require.NoError(t, err)
			require.Equal(t, "Ok", *resp.DirectiveInputType)
		})
	})
	t.Run("object field directives", func(t *testing.T) {
		t.Run("when function success", func(t *testing.T) {
			var resp struct {
				DirectiveObject *struct {
					Text         string
					NullableText *string
					Order        []string
				}
			}

			err := c.Post(`query { directiveObject{ text nullableText order} }`, &resp)

			require.NoError(t, err)
			require.Equal(t, "Ok", resp.DirectiveObject.Text)
			require.Nil(t, resp.DirectiveObject.NullableText)
			require.Equal(t, "Query_field", resp.DirectiveObject.Order[0])
			require.Equal(t, "order2_1", resp.DirectiveObject.Order[1])
			require.Equal(t, "order1_2", resp.DirectiveObject.Order[2])
			require.Equal(t, "order1_1", resp.DirectiveObject.Order[3])
		})
		t.Run("when directive returns nil & custom go field is not nilable", func(t *testing.T) {
			var resp struct {
				DirectiveObjectWithCustomGoModel *struct {
					NullableText *string
				}
			}

			err := c.Post(`query { directiveObjectWithCustomGoModel{ nullableText } }`, &resp)

			require.NoError(t, err)
			require.Nil(t, resp.DirectiveObjectWithCustomGoModel.NullableText)
		})
	})

	t.Run("Subscription directives", func(t *testing.T) {
		t.Run("arg directives", func(t *testing.T) {
			t.Run("when function errors on directives", func(t *testing.T) {
				var resp struct {
					DirectiveArg *string
				}

				err := c.WebsocketOnce(`subscription { directiveArg(arg: "") }`, &resp)

				require.EqualError(t, err, `[{"message":"invalid length","path":["directiveArg","arg"]}]`)
				require.Nil(t, resp.DirectiveArg)
			})
			t.Run("when function errors on nullable arg directives", func(t *testing.T) {
				var resp struct {
					DirectiveNullableArg *string
				}

				err := c.WebsocketOnce(`subscription { directiveNullableArg(arg: -100) }`, &resp)

				require.EqualError(t, err, `[{"message":"too small","path":["directiveNullableArg","arg"]}]`)
				require.Nil(t, resp.DirectiveNullableArg)
			})
			t.Run("when function success on nullable arg directives", func(t *testing.T) {
				var resp struct {
					DirectiveNullableArg *string
				}

				err := c.WebsocketOnce(`subscription { directiveNullableArg }`, &resp)

				require.NoError(t, err)
				require.Equal(t, "Ok", *resp.DirectiveNullableArg)
			})
			t.Run("when function success on valid nullable arg directives", func(t *testing.T) {
				var resp struct {
					DirectiveNullableArg *string
				}

				err := c.WebsocketOnce(`subscription { directiveNullableArg(arg: 1) }`, &resp)

				require.NoError(t, err)
				require.Equal(t, "Ok", *resp.DirectiveNullableArg)
			})
			t.Run("when function success", func(t *testing.T) {
				var resp struct {
					DirectiveArg *string
				}

				err := c.WebsocketOnce(`subscription { directiveArg(arg: "test") }`, &resp)

				require.NoError(t, err)
				require.Equal(t, "Ok", *resp.DirectiveArg)
			})
		})
	})
}

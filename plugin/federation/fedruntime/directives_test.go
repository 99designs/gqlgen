package fedruntime

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainDirectives_NoDirectives(t *testing.T) {
	ctx := context.Background()
	base := func(ctx context.Context) (any, error) {
		return "base result", nil
	}

	result, err := ChainDirectives(ctx, base, nil)
	require.NoError(t, err)
	assert.Equal(t, "base result", result)
}

func TestChainDirectives_SingleDirective(t *testing.T) {
	ctx := context.Background()
	base := func(ctx context.Context) (any, error) {
		return "base", nil
	}

	directives := []DirectiveFunc{
		func(ctx context.Context, next ResolverFunc) (any, error) {
			result, err := next(ctx)
			if err != nil {
				return nil, err
			}
			return fmt.Sprintf("directive1(%v)", result), nil
		},
	}

	result, err := ChainDirectives(ctx, base, directives)
	require.NoError(t, err)
	assert.Equal(t, "directive1(base)", result)
}

func TestChainDirectives_MultipleDirectives(t *testing.T) {
	ctx := context.Background()
	base := func(ctx context.Context) (any, error) {
		return "base", nil
	}

	// Directives are applied in order: directive1 wraps directive2 wraps base
	directives := []DirectiveFunc{
		func(ctx context.Context, next ResolverFunc) (any, error) {
			result, err := next(ctx)
			if err != nil {
				return nil, err
			}
			return fmt.Sprintf("d1(%v)", result), nil
		},
		func(ctx context.Context, next ResolverFunc) (any, error) {
			result, err := next(ctx)
			if err != nil {
				return nil, err
			}
			return fmt.Sprintf("d2(%v)", result), nil
		},
		func(ctx context.Context, next ResolverFunc) (any, error) {
			result, err := next(ctx)
			if err != nil {
				return nil, err
			}
			return fmt.Sprintf("d3(%v)", result), nil
		},
	}

	result, err := ChainDirectives(ctx, base, directives)
	require.NoError(t, err)
	// Execution order: d1 -> d2 -> d3 -> base
	// Result builds up: base -> d3(base) -> d2(d3(base)) -> d1(d2(d3(base)))
	assert.Equal(t, "d1(d2(d3(base)))", result)
}

func TestChainDirectives_ErrorPropagation(t *testing.T) {
	ctx := context.Background()

	t.Run("error from base", func(t *testing.T) {
		base := func(ctx context.Context) (any, error) {
			return nil, errors.New("base error")
		}

		directives := []DirectiveFunc{
			func(ctx context.Context, next ResolverFunc) (any, error) {
				return next(ctx)
			},
		}

		result, err := ChainDirectives(ctx, base, directives)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "base error", err.Error())
	})

	t.Run("error from directive", func(t *testing.T) {
		base := func(ctx context.Context) (any, error) {
			return "base", nil
		}

		directives := []DirectiveFunc{
			func(ctx context.Context, next ResolverFunc) (any, error) {
				return next(ctx)
			},
			func(ctx context.Context, next ResolverFunc) (any, error) {
				return nil, errors.New("directive error")
			},
		}

		result, err := ChainDirectives(ctx, base, directives)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "directive error", err.Error())
	})
}

func TestChainDirectives_ContextPropagation(t *testing.T) {
	type contextKey string
	key := contextKey("test-key")

	ctx := context.WithValue(context.Background(), key, "test-value")
	base := func(ctx context.Context) (any, error) {
		val := ctx.Value(key)
		return val, nil
	}

	directives := []DirectiveFunc{
		func(ctx context.Context, next ResolverFunc) (any, error) {
			// Verify context is passed through
			assert.Equal(t, "test-value", ctx.Value(key))
			return next(ctx)
		},
	}

	result, err := ChainDirectives(ctx, base, directives)
	require.NoError(t, err)
	assert.Equal(t, "test-value", result)
}

func TestChainDirectives_DirectiveModifiesContext(t *testing.T) {
	type contextKey string
	key := contextKey("counter")

	ctx := context.WithValue(context.Background(), key, 0)
	base := func(ctx context.Context) (any, error) {
		val := ctx.Value(key)
		return val, nil
	}

	directives := []DirectiveFunc{
		func(ctx context.Context, next ResolverFunc) (any, error) {
			// Directive 1 increments counter
			currentVal := ctx.Value(key).(int)
			ctx = context.WithValue(ctx, key, currentVal+1)
			return next(ctx)
		},
		func(ctx context.Context, next ResolverFunc) (any, error) {
			// Directive 2 increments counter
			currentVal := ctx.Value(key).(int)
			ctx = context.WithValue(ctx, key, currentVal+1)
			return next(ctx)
		},
	}

	result, err := ChainDirectives(ctx, base, directives)
	require.NoError(t, err)
	// Both directives increment, so base should see 2
	assert.Equal(t, 2, result)
}

func TestChainDirectives_RealWorldAuthExample(t *testing.T) {
	type contextKey string
	userKey := contextKey("user")

	t.Run("authenticated request", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userKey, "alice")
		base := func(ctx context.Context) (any, error) {
			return map[string]string{"user": "alice", "email": "alice@example.com"}, nil
		}

		// Simulate @auth directive
		authDirective := func(ctx context.Context, next ResolverFunc) (any, error) {
			user := ctx.Value(userKey)
			if user == nil {
				return nil, errors.New("unauthorized")
			}
			return next(ctx)
		}

		// Simulate @log directive
		logDirective := func(ctx context.Context, next ResolverFunc) (any, error) {
			t.Logf("@log: before resolver")
			result, err := next(ctx)
			t.Logf("@log: after resolver, result=%v, err=%v", result, err)
			return result, err
		}

		directives := []DirectiveFunc{authDirective, logDirective}

		result, err := ChainDirectives(ctx, base, directives)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("unauthenticated request blocked", func(t *testing.T) {
		ctx := context.Background() // No user in context
		base := func(ctx context.Context) (any, error) {
			t.Fatal("base should not be called")
			return nil, nil
		}

		authDirective := func(ctx context.Context, next ResolverFunc) (any, error) {
			user := ctx.Value(userKey)
			if user == nil {
				return nil, errors.New("unauthorized")
			}
			return next(ctx)
		}

		directives := []DirectiveFunc{authDirective}

		result, err := ChainDirectives(ctx, base, directives)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "unauthorized", err.Error())
	})
}

func TestWrapEntityResolver_NoDirectives(t *testing.T) {
	ctx := context.Background()
	resolver := func(ctx context.Context) (string, error) {
		return "result", nil
	}

	result, err := WrapEntityResolver(ctx, resolver, nil)
	require.NoError(t, err)
	assert.Equal(t, "result", result)
}

func TestWrapEntityResolver_WithDirectives(t *testing.T) {
	ctx := context.Background()
	resolver := func(ctx context.Context) (string, error) {
		return "base", nil
	}

	directive := func(ctx context.Context, next ResolverFunc) (any, error) {
		result, err := next(ctx)
		if err != nil {
			return nil, err
		}
		return result.(string) + " + wrapped", nil
	}

	result, err := WrapEntityResolver(ctx, resolver, []DirectiveFunc{directive})
	require.NoError(t, err)
	assert.Equal(t, "base + wrapped", result)
}

func TestWrapEntityResolver_TypeAssertion(t *testing.T) {
	ctx := context.Background()

	t.Run("correct type returned", func(t *testing.T) {
		type User struct {
			ID   string
			Name string
		}

		resolver := func(ctx context.Context) (*User, error) {
			return &User{ID: "1", Name: "Alice"}, nil
		}

		directive := func(ctx context.Context, next ResolverFunc) (any, error) {
			result, err := next(ctx)
			if err != nil {
				return nil, err
			}
			user := result.(*User)
			user.Name += " (modified)"
			return user, nil
		}

		result, err := WrapEntityResolver(ctx, resolver, []DirectiveFunc{directive})
		require.NoError(t, err)
		assert.Equal(t, "Alice (modified)", result.Name)
	})

	t.Run("wrong type from directive", func(t *testing.T) {
		resolver := func(ctx context.Context) (string, error) {
			return "correct type", nil
		}

		// Directive returns wrong type
		directive := func(ctx context.Context, next ResolverFunc) (any, error) {
			_, err := next(ctx)
			if err != nil {
				return nil, err
			}
			return 123, nil // wrong type - should be string
		}

		_, err := WrapEntityResolver(ctx, resolver, []DirectiveFunc{directive})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected type")
	})
}

func TestWrapEntityResolver_ErrorPropagation(t *testing.T) {
	ctx := context.Background()

	t.Run("resolver error", func(t *testing.T) {
		expectedErr := errors.New("resolver error")
		resolver := func(ctx context.Context) (string, error) {
			return "", expectedErr
		}

		result, err := WrapEntityResolver(ctx, resolver, nil)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, result)
	})

	t.Run("directive error", func(t *testing.T) {
		resolver := func(ctx context.Context) (string, error) {
			return "result", nil
		}

		expectedErr := errors.New("directive error")
		directive := func(ctx context.Context, next ResolverFunc) (any, error) {
			return nil, expectedErr
		}

		result, err := WrapEntityResolver(ctx, resolver, []DirectiveFunc{directive})
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Empty(t, result)
	})
}

func TestWrapEntityResolver_ComplexTypes(t *testing.T) {
	type EmailHost struct {
		ID     string
		Domain string
	}

	t.Run("pointer to struct", func(t *testing.T) {
		ctx := context.Background()
		resolver := func(ctx context.Context) (*EmailHost, error) {
			return &EmailHost{ID: "1", Domain: "example.com"}, nil
		}

		result, err := WrapEntityResolver(ctx, resolver, nil)
		require.NoError(t, err)
		assert.Equal(t, "1", result.ID)
		assert.Equal(t, "example.com", result.Domain)
	})

	t.Run("interface type", func(t *testing.T) {
		ctx := context.Background()
		resolver := func(ctx context.Context) (any, error) {
			return map[string]string{"key": "value"}, nil
		}

		result, err := WrapEntityResolver(ctx, resolver, nil)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"key": "value"}, result)
	})

	t.Run("slice type", func(t *testing.T) {
		ctx := context.Background()
		resolver := func(ctx context.Context) ([]string, error) {
			return []string{"a", "b", "c"}, nil
		}

		result, err := WrapEntityResolver(ctx, resolver, nil)
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})
}

func TestValidateEntityKeys_SingleResolver(t *testing.T) {
	t.Run("valid single key field", func(t *testing.T) {
		rep := map[string]any{"id": "123"}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findUserByID",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"id"}},
				},
			},
		}

		resolverName, err := ValidateEntityKeys("User", rep, checks)
		require.NoError(t, err)
		assert.Equal(t, "findUserByID", resolverName)
	})

	t.Run("missing key field", func(t *testing.T) {
		rep := map[string]any{"name": "Alice"}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findUserByID",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"id"}},
				},
			},
		}

		_, err := ValidateEntityKeys("User", rep, checks)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing Key Field")
	})

	t.Run("null key field", func(t *testing.T) {
		rep := map[string]any{"id": nil}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findUserByID",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"id"}},
				},
			},
		}

		_, err := ValidateEntityKeys("User", rep, checks)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "all null value KeyFields")
	})
}

func TestValidateEntityKeys_MultipleResolvers(t *testing.T) {
	t.Run("first resolver matches", func(t *testing.T) {
		rep := map[string]any{"id": "123"}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findUserByID",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"id"}},
				},
			},
			{
				ResolverName: "findUserByEmail",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"email"}},
				},
			},
		}

		resolverName, err := ValidateEntityKeys("User", rep, checks)
		require.NoError(t, err)
		assert.Equal(t, "findUserByID", resolverName)
	})

	t.Run("second resolver matches", func(t *testing.T) {
		rep := map[string]any{"email": "alice@example.com"}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findUserByID",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"id"}},
				},
			},
			{
				ResolverName: "findUserByEmail",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"email"}},
				},
			},
		}

		resolverName, err := ValidateEntityKeys("User", rep, checks)
		require.NoError(t, err)
		assert.Equal(t, "findUserByEmail", resolverName)
	})

	t.Run("no resolver matches", func(t *testing.T) {
		rep := map[string]any{"name": "Alice"}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findUserByID",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"id"}},
				},
			},
			{
				ResolverName: "findUserByEmail",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"email"}},
				},
			},
		}

		_, err := ValidateEntityKeys("User", rep, checks)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "type not found")
	})
}

func TestValidateEntityKeys_NestedFields(t *testing.T) {
	t.Run("valid nested field", func(t *testing.T) {
		rep := map[string]any{
			"user": map[string]any{
				"id": "123",
			},
		}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findByUserID",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"user", "id"}},
				},
			},
		}

		resolverName, err := ValidateEntityKeys("Review", rep, checks)
		require.NoError(t, err)
		assert.Equal(t, "findByUserID", resolverName)
	})

	t.Run("missing nested field", func(t *testing.T) {
		rep := map[string]any{
			"user": map[string]any{
				"name": "Alice",
			},
		}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findByUserID",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"user", "id"}},
				},
			},
		}

		_, err := ValidateEntityKeys("Review", rep, checks)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing Key Field")
	})

	t.Run("intermediate field not a map", func(t *testing.T) {
		rep := map[string]any{
			"user": "not-a-map",
		}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findByUserID",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"user", "id"}},
				},
			},
		}

		_, err := ValidateEntityKeys("Review", rep, checks)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not matching map[string]any")
	})
}

func TestValidateEntityKeys_MultipleKeyFields(t *testing.T) {
	t.Run("all key fields present", func(t *testing.T) {
		rep := map[string]any{
			"manufacturerID": "mfg-123",
			"productID":      "prod-456",
		}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findProductByManufacturerAndProduct",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"manufacturerID"}},
					{FieldPath: []string{"productID"}},
				},
			},
		}

		resolverName, err := ValidateEntityKeys("Product", rep, checks)
		require.NoError(t, err)
		assert.Equal(t, "findProductByManufacturerAndProduct", resolverName)
	})

	t.Run("one key field missing", func(t *testing.T) {
		rep := map[string]any{
			"manufacturerID": "mfg-123",
		}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findProductByManufacturerAndProduct",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"manufacturerID"}},
					{FieldPath: []string{"productID"}},
				},
			},
		}

		_, err := ValidateEntityKeys("Product", rep, checks)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing Key Field")
	})

	t.Run("all key fields null", func(t *testing.T) {
		rep := map[string]any{
			"manufacturerID": nil,
			"productID":      nil,
		}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findProductByManufacturerAndProduct",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"manufacturerID"}},
					{FieldPath: []string{"productID"}},
				},
			},
		}

		_, err := ValidateEntityKeys("Product", rep, checks)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "all null value KeyFields")
	})

	t.Run("some key fields null but not all", func(t *testing.T) {
		rep := map[string]any{
			"manufacturerID": "mfg-123",
			"productID":      nil,
		}
		checks := []ResolverKeyCheck{
			{
				ResolverName: "findProductByManufacturerAndProduct",
				KeyFields: []KeyFieldCheck{
					{FieldPath: []string{"manufacturerID"}},
					{FieldPath: []string{"productID"}},
				},
			},
		}

		resolverName, err := ValidateEntityKeys("Product", rep, checks)
		require.NoError(t, err)
		assert.Equal(t, "findProductByManufacturerAndProduct", resolverName)
	})
}

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

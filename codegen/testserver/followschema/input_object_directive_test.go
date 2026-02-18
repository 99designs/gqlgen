package followschema

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
)

// TestInputObjectDirectiveBug demonstrates the bug where INPUT_OBJECT directives
// are incorrectly called multiple times - once for each field that references the type.
//
// Related to: https://github.com/99designs/gqlgen/issues/2281
//
// The @directive3 is defined as: directive @directive3 on INPUT_OBJECT
// It is applied to: input InputDirectives @directive3 { ... }
//
// Expected: @directive3 should be called ONCE when InputDirectives is unmarshaled
// Actual Bug: @directive3 is called MULTIPLE TIMES - once for each field of type InputDirectives
func TestInputObjectDirectiveBug(t *testing.T) {
	resolvers := &Stub{}
	ok := "Ok"

	// Track how many times directive3 is called and with what objects
	var directive3CallCount atomic.Int32
	var directive3Objects []any

	resolvers.QueryResolver.DirectiveInput = func(ctx context.Context, arg InputDirectives) (*string, error) {
		return &ok, nil
	}

	resolvers.QueryResolver.DirectiveInputNullable = func(ctx context.Context, arg *InputDirectives) (*string, error) {
		return &ok, nil
	}

	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			// directive3 is on INPUT_OBJECT location
			Directive3: func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
				directive3CallCount.Add(1)
				directive3Objects = append(directive3Objects, obj)
				t.Logf("directive3 called with obj type: %T, value: %+v", obj, obj)
				return next(ctx)
			},
			// Length directive is on INPUT_FIELD_DEFINITION - just pass through
			Length: func(ctx context.Context, obj any, next graphql.Resolver, min int, max *int, message *string) (any, error) {
				return next(ctx)
			},
		},
	}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("single input with @directive3 should call directive once", func(t *testing.T) {
		directive3CallCount.Store(0)
		directive3Objects = nil

		var resp struct {
			DirectiveInput *string
		}

		err := c.Post(`query { directiveInput(arg: {text:"test", inner:{message:"msg"}}) }`, &resp)
		require.NoError(t, err)

		callCount := directive3CallCount.Load()
		t.Logf("directive3 was called %d time(s)", callCount)
		t.Logf("directive3 objects: %+v", directive3Objects)

		// BUG: Currently this will fail because directive3 is called multiple times
		// It gets called for:
		// 1. The 'inner' field (InnerDirectives type)
		// 2. The 'innerNullable' field (even though it's not provided)
		// 3. etc.
		//
		// Expected: Should be called exactly 1 time for the InputDirectives object
		require.Equal(t, int32(1), callCount,
			"@directive3 should be called exactly once for InputDirectives, but was called %d times", callCount)
	})

	t.Run("nullable input with @directive3 should call directive once", func(t *testing.T) {
		directive3CallCount.Store(0)
		directive3Objects = nil

		var resp struct {
			DirectiveInputNullable *string
		}

		err := c.Post(`query { directiveInputNullable(arg: {text:"test", inner:{message:"msg"}}) }`, &resp)
		require.NoError(t, err)

		callCount := directive3CallCount.Load()
		t.Logf("directive3 was called %d time(s) for nullable input", callCount)

		// BUG: Same issue - called multiple times instead of once
		require.Equal(t, int32(1), callCount,
			"@directive3 should be called exactly once for nullable InputDirectives, but was called %d times", callCount)
	})
}

// TestInputObjectDirectiveCorrectObject verifies that when INPUT_OBJECT directives
// are called, they receive the correct object (the input object being unmarshaled,
// not parent objects).
func TestInputObjectDirectiveCorrectObject(t *testing.T) {
	resolvers := &Stub{}
	ok := "Ok"

	var receivedObjType string

	resolvers.QueryResolver.DirectiveInput = func(ctx context.Context, arg InputDirectives) (*string, error) {
		return &ok, nil
	}

	srv := handler.New(NewExecutableSchema(Config{
		Resolvers: resolvers,
		Directives: DirectiveRoot{
			Directive3: func(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
				receivedObjType = ""
				if obj != nil {
					receivedObjType = string(reflect.TypeOf(obj).String())
				}
				t.Logf("directive3 obj parameter type: %T", obj)
				return next(ctx)
			},
			// Length directive is on INPUT_FIELD_DEFINITION - just pass through
			Length: func(ctx context.Context, obj any, next graphql.Resolver, min int, max *int, message *string) (any, error) {
				return next(ctx)
			},
		},
	}))
	srv.AddTransport(transport.POST{})
	c := client.New(srv)

	t.Run("directive should receive the InputDirectives object, not parent", func(t *testing.T) {
		var resp struct {
			DirectiveInput *string
		}

		err := c.Post(`query { directiveInput(arg: {text:"test", inner:{message:"msg"}}) }`, &resp)
		require.NoError(t, err)

		// The directive should receive the InputDirectives struct or a map representing it
		// It should NOT receive parent query objects
		t.Logf("Received object type: %s", receivedObjType)

		// This test documents what object type is received
		// After the fix, it should be the properly unmarshaled InputDirectives
	})
}

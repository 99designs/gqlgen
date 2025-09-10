package graphql

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

type ResolveFieldTest struct {
	name                      string
	recoverFromPanic          bool
	nonNull                   bool
	initializeFieldContextErr error
	panicMiddlewareChain      string
	panicResolverMiddleware   string
	panicFieldResolver        string
	fieldResolverValue        any
	fieldResolverErr          error
	marshalCalls              []int
	expected                  string
	expectedPanic             string
	expectedErr               string
	expectedCalls             int
}

var commonResolveFieldTests = []ResolveFieldTest{
	{
		name:               "should handle nullable field when field resolver returns nil value",
		fieldResolverValue: nil,
		expected:           "null",
		expectedCalls:      4,
	},
	{
		name:               "should fail non nullable field when field resolver returns nil value",
		nonNull:            true,
		fieldResolverValue: nil,
		expected:           "null",
		expectedErr:        "input: testField must not be null\n",
		expectedCalls:      4,
	},
	{
		name:                      "should fail when initialize field context returns an error",
		initializeFieldContextErr: errors.New("test initialize field context error"),
		expected:                  "null",
		expectedCalls:             1,
	},
	{
		name:                 "should not recover from panic when middleware chain panics",
		panicMiddlewareChain: "test middleware chain panic",
		expectedPanic:        "test middleware chain panic",
		expectedCalls:        2,
	},
	{
		name:                 "should recover from panic when middleware chain panics with recover from panic",
		recoverFromPanic:     true,
		panicMiddlewareChain: "test middleware chain panic",
		expected:             "null",
		expectedPanic:        "test middleware chain panic",
		expectedCalls:        3,
	},
	{
		name:                    "should not recover from panic when resolver middleware panics",
		panicResolverMiddleware: "test resolver middleware panic",
		expectedPanic:           "test resolver middleware panic",
		expectedCalls:           3,
	},
	{
		name:                    "should recover from panic when resolver middleware panics with recover from panic",
		recoverFromPanic:        true,
		panicResolverMiddleware: "test resolver middleware panic",
		expected:                "null",
		expectedPanic:           "test resolver middleware panic",
		expectedCalls:           4,
	},
	{
		name:               "should not recover from panic when field resolver panics",
		panicFieldResolver: "test field resolver panic",
		expectedPanic:      "test field resolver panic",
		expectedCalls:      4,
	},
	{
		name:               "should recover from panic when field resolver panics with recover from panic",
		recoverFromPanic:   true,
		panicFieldResolver: "test field resolver panic",
		expected:           "null",
		expectedPanic:      "test field resolver panic",
		expectedCalls:      5,
	},
	{
		name:             "should fail when field resolver returns an error",
		fieldResolverErr: errors.New("test field resolver error"),
		expected:         "null",
		expectedErr:      "input: testField test field resolver error\n",
		expectedCalls:    4,
	},
}

func TestResolveField(t *testing.T) {
	tests := append(
		[]ResolveFieldTest{
			{
				name:               "should resolve field",
				fieldResolverValue: "test value",
				marshalCalls:       []int{5},
				expected:           `"test value"`,
				expectedCalls:      5,
			},
		},
		commonResolveFieldTests...,
	)

	testResolveField(t, tests, ResolveField, func(t *testing.T, test ResolveFieldTest, result Marshaler) {
		var sb strings.Builder
		if result != nil {
			result.MarshalGQL(&sb)
		}
		assert.Equal(t, test.expected, sb.String())
	})
}

func TestResolveFieldStream(t *testing.T) {
	resultChan := make(chan string, 3)
	resultChan <- "test one"
	resultChan <- "test two"
	resultChan <- "test three"
	close(resultChan)
	tests := append(
		[]ResolveFieldTest{
			{
				name:               "should resolve field",
				fieldResolverValue: (<-chan string)(resultChan),
				marshalCalls:       []int{5, 6, 7},
				expected:           `{"testField":"test one"}{"testField":"test two"}{"testField":"test three"}`,
				expectedCalls:      7,
			},
		},
		commonResolveFieldTests...,
	)
	for i, test := range tests {
		// the stream tests output empty string where the non stream tests output null
		if test.expected == "null" {
			tests[i].expected = ""
		}
	}

	testResolveField(t, tests, ResolveFieldStream, func(t *testing.T, test ResolveFieldTest, result func(ctx context.Context) Marshaler) {
		var sb strings.Builder
		if result != nil {
			for range 3 {
				result(context.Background()).MarshalGQL(&sb)
			}
		}
		assert.Equal(t, test.expected, sb.String())
	})
}

type resolveFieldFunc[T, R any] func(
	ctx context.Context,
	oc *OperationContext,
	field CollectedField,
	initializeFieldContext func(ctx context.Context, field CollectedField) (*FieldContext, error),
	fieldResolver func(context.Context) (any, error),
	middlewareChain func(ctx context.Context, next Resolver) Resolver,
	marshal func(ctx context.Context, sel ast.SelectionSet, v T) Marshaler,
	recoverFromPanic bool,
	nonNull bool,
) R

type resolveFieldTestKey string

func testResolveField[R any](
	t *testing.T,
	tests []ResolveFieldTest,
	resolveField resolveFieldFunc[string, R],
	assertResult func(t *testing.T, test ResolveFieldTest, result R),
) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			assertCall := func(call int) {
				calls++
				assert.Equal(t, call, calls)
			}
			marshalCalled := 0

			ctx := WithResponseContext(context.Background(), DefaultErrorPresenter, nil)
			oc := &OperationContext{
				RecoverFunc: func(ctx context.Context, err any) (userMessage error) {
					assertCall(test.expectedCalls)
					if test.expectedPanic != "" {
						assert.Equal(t, test.expectedPanic, err)
					} else {
						t.Errorf("should not panic but recover func called with: %v", err)
					}
					return nil
				},
				ResolverMiddleware: func(ctx context.Context, next Resolver) (res any, err error) {
					assertCall(3)
					ctx = context.WithValue(ctx, resolveFieldTestKey("resolver"), "middleware")
					if test.panicResolverMiddleware != "" {
						panic(test.panicResolverMiddleware)
					}
					return next(ctx)
				},
			}
			field := CollectedField{
				Field: &ast.Field{
					Alias: "testField",
				},
			}
			var result R
			run := func() {
				result = resolveField(
					ctx,
					oc,
					field,
					func(ctx context.Context, field CollectedField) (*FieldContext, error) {
						assertCall(1)
						return &FieldContext{
							Object: "Test",
							Field:  field,
						}, test.initializeFieldContextErr
					},
					func(ctx context.Context) (any, error) {
						assertCall(4)
						assert.Equal(t, "middleware", ctx.Value(resolveFieldTestKey("resolver")), "should propagate value from resolver middleware")
						if test.panicFieldResolver != "" {
							panic(test.panicFieldResolver)
						}
						return test.fieldResolverValue, test.fieldResolverErr
					},
					func(ctx context.Context, next Resolver) Resolver {
						assertCall(2)
						if test.panicMiddlewareChain != "" {
							panic(test.panicMiddlewareChain)
						}
						return next
					},
					func(ctx context.Context, sel ast.SelectionSet, v string) Marshaler {
						assertCall(test.marshalCalls[marshalCalled])
						marshalCalled++
						return MarshalString(v)
					},
					test.recoverFromPanic,
					test.nonNull,
				)
			}
			if test.expectedPanic == "" || test.recoverFromPanic {
				run()
			} else {
				assert.PanicsWithValue(t, test.expectedPanic, run)
			}
			require.EqualError(t, GetErrors(ctx), test.expectedErr)

			assertResult(t, test, result)
			assert.Equal(t, test.expectedCalls, calls)
		})
	}
}

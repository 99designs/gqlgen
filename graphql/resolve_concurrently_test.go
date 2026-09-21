package graphql

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func newResolveConcurrentlyTestContext() (context.Context, *OperationContext) {
	ctx := WithResponseContext(context.Background(), DefaultErrorPresenter, nil)
	return ctx, &OperationContext{
		RecoverFunc: DefaultRecover,
		RootResolverMiddleware: func(ctx context.Context, next RootResolver) Marshaler {
			return next(ctx)
		},
	}
}

func collectedField(alias string, deferLabels ...string) CollectedField {
	field := CollectedField{Field: &ast.Field{Alias: alias, Name: alias}}
	for _, label := range deferLabels {
		field.Deferrables = append(field.Deferrables, &Deferrable{Label: label})
	}
	return field
}

// scheduleFunc schedules a test's single field on out through one of the two
// helpers under test.
type scheduleFunc func(
	ctx context.Context,
	oc *OperationContext,
	out *FieldSet,
	test resolveConcurrentlyTest,
)

type resolveConcurrentlyTest struct {
	name             string
	recoverFromPanic bool
	nonNull          bool
	resolve          func(context.Context) Marshaler
	wantJSON         string
	wantInvalids     uint32
	wantErrs         string
	wantPanic        any
}

// scalarResolveTests cover the per-field bookkeeping both helpers share: the
// invalids sentinel and the panic guard.
var scalarResolveTests = []resolveConcurrentlyTest{
	{
		name:     "resolves the field",
		resolve:  func(context.Context) Marshaler { return MarshalString("bob") },
		wantJSON: `{"name":"bob"}`,
	},
	{
		name:         "counts a null non-null field as invalid",
		nonNull:      true,
		resolve:      func(context.Context) Marshaler { return Null },
		wantJSON:     `{"name":null}`,
		wantInvalids: 1,
	},
	{
		name:         "counts a required null nullable field as invalid",
		resolve:      func(context.Context) Marshaler { return RequiredNull },
		wantJSON:     `{"name":null}`,
		wantInvalids: 1,
	},
	{
		name:     "accepts a null nullable field",
		resolve:  func(context.Context) Marshaler { return Null },
		wantJSON: `{"name":null}`,
	},
	{
		name:             "reports a panic when recovering is enabled",
		recoverFromPanic: true,
		nonNull:          true,
		resolve:          func(context.Context) Marshaler { panic("boom") },
		wantErrs:         "input: internal system error\n",
	},
	{
		name:      "propagates a panic when recovering is disabled",
		nonNull:   true,
		resolve:   func(context.Context) Marshaler { panic("boom") },
		wantPanic: "boom",
	},
}

func scheduleConcurrently(
	ctx context.Context,
	oc *OperationContext,
	out *FieldSet,
	test resolveConcurrentlyTest,
) {
	deferred := NewDeferredGroup(ctx)
	out.ResolveConcurrently(oc, &deferred, 0, test.recoverFromPanic, test.nonNull, test.resolve)
}

func scheduleRootConcurrently(
	ctx context.Context,
	oc *OperationContext,
	out *FieldSet,
	test resolveConcurrentlyTest,
) {
	out.ResolveRootConcurrently(ctx, oc, 0, test.recoverFromPanic, test.nonNull, test.resolve)
}

// runScalarResolveTest schedules test's field through schedule, dispatches, and
// checks the marshalled output, the invalids counter and the errors.
func runScalarResolveTest(
	t *testing.T,
	test resolveConcurrentlyTest,
	schedule scheduleFunc,
) {
	t.Helper()
	ctx, oc := newResolveConcurrentlyTestContext()
	out := NewFieldSet([]CollectedField{collectedField("name")})

	schedule(ctx, oc, out, test)

	if test.wantPanic != nil {
		assert.PanicsWithValue(t, test.wantPanic, func() { out.Dispatch(ctx) })
		return
	}
	out.Dispatch(ctx)

	assert.EqualValues(t, test.wantInvalids, out.Invalids)
	if test.wantErrs != "" {
		assert.Equal(t, test.wantErrs, GetErrors(ctx).Error())
		return
	}
	assert.Empty(t, GetErrors(ctx))

	var b bytes.Buffer
	out.MarshalGQL(&b)
	assert.JSONEq(t, test.wantJSON, b.String())
}

func TestResolveConcurrently(t *testing.T) {
	t.Parallel()

	for _, test := range scalarResolveTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runScalarResolveTest(t, test, scheduleConcurrently)
		})
	}

	t.Run("moves a deferred field to the deferred field set", func(t *testing.T) {
		t.Parallel()
		ctx, oc := newResolveConcurrentlyTestContext()
		field := collectedField("name", "labelA", "labelB")
		out := NewFieldSet([]CollectedField{field})
		deferred := NewDeferredGroup(ctx)

		out.ResolveConcurrently(
			oc,
			&deferred,
			0,
			true,
			true,
			func(context.Context) Marshaler { return MarshalString("bob") },
		)

		assert.Same(t, Null, out.Values[0])
		require.Len(t, deferred.FieldSet.Values, 1)
		require.Len(t, deferred.Defers, 2)

		deferred.FieldSet.Dispatch(ctx)
		for _, label := range []string{"labelA", "labelB"} {
			assert.Equal(t, []int{0}, deferred.Defers[label].indices, label)
		}

		var b bytes.Buffer
		deferred.Defers["labelA"].MarshalGQL(&b)
		assert.JSONEq(t, `{"name":"bob"}`, b.String())
		// The value is not counted against the enclosing object.
		assert.Zero(t, out.Invalids)
	})

	t.Run("reuses one view per defer label", func(t *testing.T) {
		t.Parallel()
		ctx, oc := newResolveConcurrentlyTestContext()
		out := NewFieldSet([]CollectedField{
			collectedField("first", "label"),
			collectedField("second", "label"),
		})
		deferred := NewDeferredGroup(ctx)

		for i := range out.fields {
			out.ResolveConcurrently(
				oc,
				&deferred,
				i,
				true,
				true,
				func(context.Context) Marshaler { return MarshalString("bob") },
			)
		}

		require.Len(t, deferred.Defers, 1)
		assert.Equal(t, []int{0, 1}, deferred.Defers["label"].indices)
	})

	t.Run("ignores defer labels on a non-deferrable field", func(t *testing.T) {
		t.Parallel()
		ctx, oc := newResolveConcurrentlyTestContext()
		field := collectedField("name", "label")
		field.IsNonDeferrable = true
		out := NewFieldSet([]CollectedField{field})
		deferred := NewDeferredGroup(ctx)

		out.ResolveConcurrently(
			oc,
			&deferred,
			0,
			true,
			true,
			func(context.Context) Marshaler { return MarshalString("bob") },
		)
		out.Dispatch(ctx)

		assert.Empty(t, deferred.FieldSet.Values)
		var b bytes.Buffer
		out.MarshalGQL(&b)
		assert.JSONEq(t, `{"name":"bob"}`, b.String())
	})
}

func TestResolveRootConcurrently(t *testing.T) {
	t.Parallel()

	for _, test := range scalarResolveTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runScalarResolveTest(t, test, scheduleRootConcurrently)
		})
	}

	t.Run("runs the resolver under the root middleware with the given context", func(t *testing.T) {
		t.Parallel()
		ctx, oc := newResolveConcurrentlyTestContext()
		type ctxKey struct{}
		rootCtx := context.WithValue(ctx, ctxKey{}, "root")

		var middlewareCalls int
		oc.RootResolverMiddleware = func(ctx context.Context, next RootResolver) Marshaler {
			middlewareCalls++
			return next(ctx)
		}

		field := collectedField("todos")
		out := NewFieldSet([]CollectedField{field})
		out.ResolveRootConcurrently(
			rootCtx,
			oc,
			0,
			true,
			true,
			func(ctx context.Context) Marshaler {
				// The context Dispatch passes to the callback is ignored in
				// favour of the one carrying this field's RootFieldContext.
				return MarshalString(ctx.Value(ctxKey{}).(string))
			},
		)
		out.Dispatch(ctx)

		assert.Equal(t, 1, middlewareCalls)
		var b bytes.Buffer
		out.MarshalGQL(&b)
		assert.JSONEq(t, `{"todos":"root"}`, b.String())
	})
}

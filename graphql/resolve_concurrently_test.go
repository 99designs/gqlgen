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

// marshalString returns a resolver that yields s, so a case can state its
// result inline rather than through a shared fixture.
func marshalString(s string) func(context.Context) Marshaler {
	return func(context.Context) Marshaler { return MarshalString(s) }
}

// sentinelCase describes one field resolution and everything the helpers are
// expected to record about it.
type sentinelCase struct {
	nonNull      bool
	resolve      func(context.Context) Marshaler
	wantJSON     string
	wantInvalids uint32
}

// sentinelCases returns the per-field bookkeeping both helpers share. Which
// sentinel invalidates the enclosing object depends on nullability: Null does
// for a non-null field, RequiredNull for a nullable one. Nothing downstream
// recomputes this, so a swapped comparison would quietly turn a null-bubbling
// error into a successful response containing null.
//
// Both helpers run the same cases so the two routes into that bookkeeping are
// comparable. Each test iterates them and asserts in its own body: the cases
// are shared, the assertions are not.
func sentinelCases() map[string]sentinelCase {
	return map[string]sentinelCase{
		"resolves the field": {
			resolve:  marshalString("bob"),
			wantJSON: `{"name":"bob"}`,
		},
		"counts a null non-null field as invalid": {
			nonNull:      true,
			resolve:      func(context.Context) Marshaler { return Null },
			wantJSON:     `{"name":null}`,
			wantInvalids: 1,
		},
		"counts a required null nullable field as invalid": {
			resolve:      func(context.Context) Marshaler { return RequiredNull },
			wantJSON:     `{"name":null}`,
			wantInvalids: 1,
		},
		"accepts a null nullable field": {
			resolve:  func(context.Context) Marshaler { return Null },
			wantJSON: `{"name":null}`,
		},
	}
}

func TestResolveConcurrently(t *testing.T) {
	t.Parallel()

	for name, tc := range sentinelCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, oc := newResolveConcurrentlyTestContext()
			out := NewFieldSet([]CollectedField{collectedField("name")})
			deferred := NewDeferredGroup(ctx)

			out.ResolveConcurrently(oc, &deferred, 0, true, tc.nonNull, tc.resolve)
			out.Dispatch(ctx)

			assert.Equal(t, tc.wantInvalids, out.Invalids)
			assert.Empty(t, GetErrors(ctx))

			var b bytes.Buffer
			out.MarshalGQL(&b)
			assert.JSONEq(t, tc.wantJSON, b.String())
		})
	}

	t.Run("moves a deferred field to the deferred field set", func(t *testing.T) {
		t.Parallel()

		ctx, oc := newResolveConcurrentlyTestContext()
		out := NewFieldSet([]CollectedField{collectedField("name", "labelA", "labelB")})
		deferred := NewDeferredGroup(ctx)

		out.ResolveConcurrently(oc, &deferred, 0, true, true, marshalString("bob"))

		assert.Same(t, Null, out.Values[0], "the initial response reports a deferred field as null")
		require.Len(t, deferred.FieldSet.Values, 1)
		require.Len(t, deferred.Defers, 2, "a field under two labels is registered under both")

		deferred.FieldSet.Dispatch(ctx)

		var b bytes.Buffer
		deferred.Defers["labelA"].MarshalGQL(&b)
		assert.JSONEq(t, `{"name":"bob"}`, b.String())

		// The value is not counted against the enclosing object: it is absent
		// from the initial response by design, not invalid.
		assert.Zero(t, out.Invalids)
	})

	t.Run("reuses one view per defer label", func(t *testing.T) {
		t.Parallel()

		ctx, oc := newResolveConcurrentlyTestContext()
		fields := []CollectedField{
			collectedField("first", "label"),
			collectedField("second", "label"),
		}
		out := NewFieldSet(fields)
		deferred := NewDeferredGroup(ctx)

		for i, field := range fields {
			out.ResolveConcurrently(oc, &deferred, i, true, true, marshalString(field.Alias))
		}

		require.Len(t, deferred.Defers, 1, "one label should yield one view, not one per field")
		deferred.FieldSet.Dispatch(ctx)

		// Both fields under the one label belong to the one incremental
		// payload. Asserting the payload rather than the view's indices keeps
		// this test on the behaviour a client observes.
		var b bytes.Buffer
		deferred.Defers["label"].MarshalGQL(&b)
		assert.JSONEq(t, `{"first":"first","second":"second"}`, b.String())
	})

	t.Run("ignores defer labels on a non-deferrable field", func(t *testing.T) {
		t.Parallel()

		ctx, oc := newResolveConcurrentlyTestContext()
		field := collectedField("name", "label")
		field.IsNonDeferrable = true
		out := NewFieldSet([]CollectedField{field})
		deferred := NewDeferredGroup(ctx)

		out.ResolveConcurrently(oc, &deferred, 0, true, true, marshalString("bob"))
		out.Dispatch(ctx)

		assert.Empty(t, deferred.FieldSet.Values)
		var b bytes.Buffer
		out.MarshalGQL(&b)
		assert.JSONEq(t, `{"name":"bob"}`, b.String())
	})
}

func TestResolveRootConcurrently(t *testing.T) {
	t.Parallel()

	// A root field reaches the same bookkeeping, but through
	// RootResolverMiddleware rather than directly.
	for name, tc := range sentinelCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, oc := newResolveConcurrentlyTestContext()
			out := NewFieldSet([]CollectedField{collectedField("name")})

			out.ResolveRootConcurrently(ctx, oc, 0, true, tc.nonNull, tc.resolve)
			out.Dispatch(ctx)

			assert.Equal(t, tc.wantInvalids, out.Invalids)
			assert.Empty(t, GetErrors(ctx))

			var b bytes.Buffer
			out.MarshalGQL(&b)
			assert.JSONEq(t, tc.wantJSON, b.String())
		})
	}

	// Synchronous root fields count the value RootResolverMiddleware returns,
	// since that is what lands in the slot and is marshalled. Concurrent root
	// fields must too, or a middleware that nulls a non-null field leaves the
	// object unbubbled.
	middlewareCases := map[string]struct {
		resolve      func(context.Context) Marshaler
		middleware   Marshaler
		wantJSON     string
		wantInvalids uint32
	}{
		"middleware nulls a resolved non-null field": {
			resolve:      marshalString("bob"),
			middleware:   Null,
			wantJSON:     `{"name":null}`,
			wantInvalids: 1,
		},
		"middleware replaces a null non-null field": {
			resolve:    func(context.Context) Marshaler { return Null },
			middleware: MarshalString("substitute"),
			wantJSON:   `{"name":"substitute"}`,
		},
	}
	for name, tc := range middlewareCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, oc := newResolveConcurrentlyTestContext()
			oc.RootResolverMiddleware = func(ctx context.Context, next RootResolver) Marshaler {
				next(ctx)
				return tc.middleware
			}
			out := NewFieldSet([]CollectedField{collectedField("name")})

			out.ResolveRootConcurrently(ctx, oc, 0, true, true, tc.resolve)
			out.Dispatch(ctx)

			assert.Equal(t, tc.wantInvalids, out.Invalids)
			var b bytes.Buffer
			out.MarshalGQL(&b)
			assert.JSONEq(t, tc.wantJSON, b.String())
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

		out := NewFieldSet([]CollectedField{collectedField("todos")})
		out.ResolveRootConcurrently(rootCtx, oc, 0, true, true,
			func(ctx context.Context) Marshaler {
				// The context Dispatch passes to the callback is ignored in
				// favour of the one carrying this field's RootFieldContext.
				return MarshalString(ctx.Value(ctxKey{}).(string))
			})
		out.Dispatch(ctx)

		assert.Equal(t, 1, middlewareCalls)
		var b bytes.Buffer
		out.MarshalGQL(&b)
		assert.JSONEq(t, `{"todos":"root"}`, b.String())
	})
}

// A recovered panic must leave behind what a resolver returning an error
// would have: the field null, counted against Invalids when the field is
// non-null so the enclosing object bubbles to null. A deferred field is held
// to the same on the deferred field set, whose slot its incremental payload
// marshals. Left nil instead, the slot would crash the object marshaller,
// which writes every entry, or vanish from the payload, which skips nils.
//
// This guard is the safety net at the goroutine boundary rather than the
// first line of defence: generated code recovers resolver panics in
// ResolveField and argument coercion panics in its fieldContext functions,
// so none reach it today. It keeps a panic that escapes them from crashing
// the process out of a Dispatch goroutine.
func TestResolveConcurrentlyNullsAFieldAfterARecoveredPanic(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		deferred     bool
		nonNull      bool
		wantInvalids uint32
	}{
		"non-null field":          {nonNull: true, wantInvalids: 1},
		"nullable field":          {},
		"deferred non-null field": {deferred: true, nonNull: true, wantInvalids: 1},
		"deferred nullable field": {deferred: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, oc := newResolveConcurrentlyTestContext()
			var labels []string
			if tc.deferred {
				labels = []string{"label"}
			}
			out := NewFieldSet([]CollectedField{collectedField("name", labels...)})
			deferred := NewDeferredGroup(ctx)

			out.ResolveConcurrently(oc, &deferred, 0, true, tc.nonNull,
				func(context.Context) Marshaler { panic("boom") })
			out.Dispatch(ctx)
			deferred.FieldSet.Dispatch(ctx)

			resolved, payload := out, Marshaler(out)
			if tc.deferred {
				resolved, payload = deferred.FieldSet, deferred.Defers["label"]
				assert.Zero(t, out.Invalids, "a deferred field is absent, not invalid")
			}

			assert.Equal(t, "input: internal system error\n", GetErrors(ctx).Error())
			require.Len(t, resolved.Values, 1)
			assert.Same(t, Null, resolved.Values[0], "the panicking field resolves to null")
			assert.Equal(t, tc.wantInvalids, resolved.Invalids)

			var b bytes.Buffer
			payload.MarshalGQL(&b)
			assert.JSONEq(t, `{"name":null}`, b.String())
		})
	}
}

// The root helper reaches the same guard through RootResolverMiddleware, so a
// panic has to survive the middleware to be reported. A middleware that
// recovered would turn this into a silent success.
func TestResolveRootConcurrentlyReportsAPanicRaisedUnderTheMiddleware(t *testing.T) {
	t.Parallel()

	ctx, oc := newResolveConcurrentlyTestContext()
	out := NewFieldSet([]CollectedField{collectedField("name")})

	out.ResolveRootConcurrently(ctx, oc, 0, true, true,
		func(context.Context) Marshaler { panic("boom") })
	out.Dispatch(ctx)

	assert.Equal(t, "input: internal system error\n", GetErrors(ctx).Error())
}

// With recovery disabled the panic reaches the caller unchanged. Generated
// code takes this path when OmitPanicHandler is set, which is a deliberate
// choice to let the process crash rather than absorb a bug.
func TestResolveConcurrentlyPropagatesAPanicWhenRecoveryIsDisabled(t *testing.T) {
	t.Parallel()

	ctx, oc := newResolveConcurrentlyTestContext()
	out := NewFieldSet([]CollectedField{collectedField("name")})
	deferred := NewDeferredGroup(ctx)

	out.ResolveConcurrently(oc, &deferred, 0, false, true,
		func(context.Context) Marshaler { panic("boom") })

	assert.PanicsWithValue(t, "boom", func() { out.Dispatch(ctx) })
}

func TestResolveRootConcurrentlyPropagatesAPanicWhenRecoveryIsDisabled(t *testing.T) {
	t.Parallel()

	ctx, oc := newResolveConcurrentlyTestContext()
	out := NewFieldSet([]CollectedField{collectedField("name")})

	out.ResolveRootConcurrently(ctx, oc, 0, false, true,
		func(context.Context) Marshaler { panic("boom") })

	assert.PanicsWithValue(t, "boom", func() { out.Dispatch(ctx) })
}

// ProcessDeferredGroup labels every incremental payload with the group's Path
// and resolves the group's field set under its Context. Both are captured when
// the group is built rather than when it is processed, so a group built from
// the wrong context sends payloads a client cannot place in the response.
func TestNewDeferredGroup(t *testing.T) {
	t.Parallel()

	ctx := WithResponseContext(context.Background(), DefaultErrorPresenter, nil)
	ctx = WithFieldContext(ctx, &FieldContext{Field: collectedField("todos")})

	deferred := NewDeferredGroup(ctx)

	assert.Equal(t, ast.Path{ast.PathName("todos")}, deferred.Path,
		"payloads are placed at the context's path")
	assert.Equal(t, ctx, deferred.Context)
	require.NotNil(t, deferred.FieldSet)
	assert.Empty(t, deferred.FieldSet.Values, "a new group has collected no fields")
	assert.NotNil(t, deferred.Defers, "the map is ready for a deferred field to register in")
	assert.Empty(t, deferred.Defers)
}

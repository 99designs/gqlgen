package graphql

import (
	"context"
	"sync/atomic"
)

// ResolveConcurrently schedules the concurrent resolution of field i of m,
// covering the panic guard, the invalids bookkeeping and the @defer group
// accounting the generated executor needs per field.
//
// When recoverFromPanic is true a panic raised by resolve is reported through
// oc and the field is left with a nil marshaler. nonNull picks the sentinel
// that marks the enclosing object invalid: Null for non-null fields,
// RequiredNull for nullable ones.
//
// A field carrying a @defer directive is appended to deferred.FieldSet instead
// of m, and registered in deferred.Defers under each of its labels so the
// incremental payload for a label is emitted once every field under it
// resolves.
//
// Like NewView, this schedules work rather than doing it, so it must be called
// from the single goroutine that owns m and deferred, before either field set
// is dispatched.
func (m *FieldSet) ResolveConcurrently(
	oc *OperationContext,
	deferred *DeferredGroup,
	i int,
	recoverFromPanic bool,
	nonNull bool,
	resolve func(ctx context.Context) Marshaler,
) {
	field := m.fields[i]
	if field.IsDeferred() {
		// Bind the field set rather than the group: capturing deferred in the
		// closure below would push the caller's group onto the heap.
		fs := deferred.FieldSet
		fs.AddField(field)
		fieldIndex := len(fs.Values) - 1
		fs.Concurrently(fieldIndex, func(ctx context.Context) Marshaler {
			return fs.resolveAndCount(ctx, oc, recoverFromPanic, nonNull, resolve)
		})

		for _, deferrable := range field.Deferrables {
			view, ok := deferred.Defers[deferrable.Label]
			if !ok {
				view = fs.NewView()
				deferred.Defers[deferrable.Label] = view
			}
			view.AddIndices(fieldIndex)
		}

		// The value travels in a later incremental payload, so it is kept out
		// of the initial response and not scheduled on m.
		m.Values[i] = Null
		return
	}

	m.Concurrently(i, func(ctx context.Context) Marshaler {
		return m.resolveAndCount(ctx, oc, recoverFromPanic, nonNull, resolve)
	})
}

// ResolveRootConcurrently schedules the concurrent resolution of field i of m,
// a Query or Mutation field, running it under the operation's
// RootResolverMiddleware. Root fields are never deferred.
//
// ctx must be the context carrying this field's RootFieldContext. The context
// Dispatch hands to the callback is deliberately ignored so that every root
// field runs under its own RootFieldContext.
func (m *FieldSet) ResolveRootConcurrently(
	ctx context.Context,
	oc *OperationContext,
	i int,
	recoverFromPanic bool,
	nonNull bool,
	resolve func(ctx context.Context) Marshaler,
) {
	m.Concurrently(i, func(context.Context) Marshaler {
		return oc.RootResolverMiddleware(ctx, func(ctx context.Context) Marshaler {
			return m.resolveAndCount(ctx, oc, recoverFromPanic, nonNull, resolve)
		})
	})
}

// resolveAndCount runs resolve, counting the result against m.Invalids when it
// is the sentinel that makes the enclosing object invalid.
func (m *FieldSet) resolveAndCount(
	ctx context.Context,
	oc *OperationContext,
	recoverFromPanic bool,
	nonNull bool,
	resolve func(ctx context.Context) Marshaler,
) (res Marshaler) {
	if recoverFromPanic {
		defer func() {
			if r := recover(); r != nil {
				oc.Error(ctx, oc.Recover(ctx, r))
			}
		}()
	}

	invalid := Marshaler(RequiredNull)
	if nonNull {
		invalid = Null
	}

	res = resolve(ctx)
	if res == invalid {
		atomic.AddUint32(&m.Invalids, 1)
	}
	return res
}

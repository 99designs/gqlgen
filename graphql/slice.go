package graphql

import (
	"context"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

// MarshalSliceConcurrently marshals a slice of elements concurrently, writing
// each result into the returned Array. It fixes a WaitGroup deadlock that occurs
// when context cancellation prevents semaphore acquisition (see issue #4018).
//
// The marshalElement callback is called for each index and receives a context
// that already has a FieldContext with Index set. The callback should set
// FieldContext.Result and perform the actual marshaling.
//
// This function handles:
//   - Single-element optimization (no goroutine spawned)
//   - Optional worker limit via semaphore to bound concurrency
//   - Correct WaitGroup handling: wg.Add(1) only after successful semaphore
//     acquire, preventing deadlocks on context cancellation
//   - Panic recovery (unless omitPanicHandler is true)
//
// workerLimit of 0 means unlimited concurrency.
func MarshalSliceConcurrently(
	ctx context.Context,
	length int,
	workerLimit int64,
	omitPanicHandler bool,
	marshalElement func(ctx context.Context, i int) Marshaler,
) Array {
	ret := make(Array, length)
	if length == 0 {
		return ret
	}

	isLen1 := length == 1

	if isLen1 {
		i := 0
		fc := &FieldContext{
			Index: &i,
		}
		childCtx := WithFieldContext(ctx, fc)
		if omitPanicHandler {
			ret[0] = marshalElement(childCtx, 0)
		} else {
			func() {
				defer func() {
					if r := recover(); r != nil {
						AddError(childCtx, Recover(childCtx, r))
						ret = nil
					}
				}()
				ret[0] = marshalElement(childCtx, 0)
			}()
		}
		return ret
	}

	// Multiple elements: use goroutines with correct WaitGroup handling.
	var wg sync.WaitGroup
	var sm *semaphore.Weighted
	if workerLimit > 0 {
		sm = semaphore.NewWeighted(workerLimit)
	}

	// retNilFlag is used to signal from any goroutine that the result should
	// be nil (e.g. on panic). We use atomic to avoid data races.
	var retNilFlag atomic.Bool

	for i := range length {
		i := i
		fc := &FieldContext{
			Index: &i,
		}
		childCtx := WithFieldContext(ctx, fc)

		f := func(i int) {
			defer wg.Done()
			if sm != nil {
				defer sm.Release(1)
			}
			if !omitPanicHandler {
				defer func() {
					if r := recover(); r != nil {
						AddError(childCtx, Recover(childCtx, r))
						retNilFlag.Store(true)
					}
				}()
			}
			ret[i] = marshalElement(childCtx, i)
		}

		if sm != nil {
			if err := sm.Acquire(ctx, 1); err != nil {
				// Context was cancelled. Report the error but do NOT call
				// wg.Add — this is the fix for issue #4018. The old generated
				// code called wg.Add(len(v)) upfront and then skipped
				// wg.Done() here, causing a deadlock.
				AddError(childCtx, ctx.Err())
				continue
			}
		}

		// Only increment wg after successful semaphore acquire (or when no
		// semaphore is configured), ensuring wg.Wait() never blocks on
		// goroutines that were never launched.
		wg.Add(1)
		go f(i)
	}

	wg.Wait()

	if retNilFlag.Load() {
		return nil
	}
	return ret
}

package graphql

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// MarshalSliceConcurrently marshals a slice of elements concurrently, writing
// each result into the returned Array.
//
// The marshalElement callback is called for each index and receives a context
// that already has a FieldContext with Index set. The callback should set
// FieldContext.Result and perform the actual marshaling.
//
// workerLimit of 0 means unlimited concurrency.
//
// Every element of the returned Array is set, because Array.MarshalGQL
// marshals each one. An element that could not be marshalled holds Null, as
// one whose resolver returned an error would: one whose marshalElement
// panicked (recovered unless omitPanicHandler), or, with a workerLimit, one
// not yet started when ctx is cancelled, reported once as ctx.Err(). A
// nullable element thus reports null in place, and for non-null elements the
// caller's non-null check nulls the whole list.
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
						ret[0] = Null
					}
				}()
				ret[0] = marshalElement(childCtx, 0)
			}()
		}
		return ret
	}

	// Multiple elements: use goroutines.
	var wg sync.WaitGroup
	var sm *semaphore.Weighted
	if workerLimit > 0 {
		sm = semaphore.NewWeighted(workerLimit)
	}

	for i := range length {
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
						ret[i] = Null
					}
				}()
			}
			ret[i] = marshalElement(childCtx, i)
		}

		if sm != nil {
			if err := sm.Acquire(ctx, 1); err != nil {
				AddError(childCtx, ctx.Err())
				// No worker owns these slots, so they are safe to write here.
				for j := i; j < length; j++ {
					ret[j] = Null
				}
				break
			}
		}

		wg.Add(1)
		go f(i)
	}

	wg.Wait()
	return ret
}

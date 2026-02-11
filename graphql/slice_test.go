package graphql

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// withTestResponseContext sets up a minimal response context for testing.
func withTestResponseContext(ctx context.Context) context.Context {
	return WithResponseContext(ctx, func(ctx context.Context, err error) *gqlerror.Error {
		return &gqlerror.Error{Message: err.Error()}
	}, DefaultRecover)
}

func TestMarshalSliceConcurrently(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		ret := MarshalSliceConcurrently(
			ctx,
			0,
			0,
			false,
			func(ctx context.Context, i int) Marshaler {
				t.Fatal("should not be called")
				return Null
			},
		)
		assert.Empty(t, ret)
	})

	t.Run("single element runs synchronously", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		var called bool
		ret := MarshalSliceConcurrently(
			ctx,
			1,
			0,
			false,
			func(ctx context.Context, i int) Marshaler {
				called = true
				assert.Equal(t, 0, i)
				fc := GetFieldContext(ctx)
				require.NotNil(t, fc)
				assert.Equal(t, 0, *fc.Index)
				return MarshalString("hello")
			},
		)
		assert.True(t, called)
		require.Len(t, ret, 1)
		var buf bytes.Buffer
		ret[0].MarshalGQL(&buf)
		assert.Equal(t, `"hello"`, buf.String())
	})

	t.Run("multiple elements run concurrently", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		n := 10
		var callCount atomic.Int32
		ret := MarshalSliceConcurrently(
			ctx,
			n,
			0,
			false,
			func(ctx context.Context, i int) Marshaler {
				callCount.Add(1)
				fc := GetFieldContext(ctx)
				require.NotNil(t, fc)
				assert.Equal(t, i, *fc.Index)
				return MarshalString(fmt.Sprintf("item-%d", i))
			},
		)
		assert.Equal(t, int32(n), callCount.Load())
		require.Len(t, ret, n)
		for i := 0; i < n; i++ {
			var buf bytes.Buffer
			ret[i].MarshalGQL(&buf)
			assert.Equal(t, fmt.Sprintf(`"item-%d"`, i), buf.String())
		}
	})

	t.Run("worker limit bounds concurrency", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		n := 20
		var workerLimit int64 = 3
		var concurrent atomic.Int32
		var maxConcurrent atomic.Int32

		ret := MarshalSliceConcurrently(
			ctx,
			n,
			workerLimit,
			false,
			func(ctx context.Context, i int) Marshaler {
				cur := concurrent.Add(1)
				defer concurrent.Add(-1)
				// Track the maximum observed concurrency
				for {
					old := maxConcurrent.Load()
					if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
						break
					}
				}
				// Small sleep to allow concurrency to build up
				time.Sleep(time.Millisecond)
				return MarshalString(fmt.Sprintf("item-%d", i))
			},
		)

		require.Len(t, ret, n)
		assert.LessOrEqual(t, maxConcurrent.Load(), int32(workerLimit))
	})

	t.Run("panic recovery sets result to nil", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		ret := MarshalSliceConcurrently(
			ctx,
			1,
			0,
			false,
			func(ctx context.Context, i int) Marshaler {
				panic("test panic")
			},
		)
		assert.Nil(t, ret)
	})

	t.Run("panic recovery in concurrent mode sets result to nil", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		ret := MarshalSliceConcurrently(
			ctx,
			3,
			0,
			false,
			func(ctx context.Context, i int) Marshaler {
				if i == 1 {
					panic("test panic")
				}
				return MarshalString("ok")
			},
		)
		assert.Nil(t, ret)
	})

	t.Run("omit panic handler does not recover", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		assert.Panics(t, func() {
			MarshalSliceConcurrently(ctx, 1, 0, true, func(ctx context.Context, i int) Marshaler {
				panic("test panic")
			})
		})
	})

	t.Run("context cancellation with worker limit does not deadlock", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		ctx, cancel := context.WithCancel(ctx)

		done := make(chan Array, 1)
		go func() {
			ret := MarshalSliceConcurrently(
				ctx,
				100,
				1,
				false,
				func(ctx context.Context, i int) Marshaler {
					if i == 2 {
						// Cancel context mid-flight to trigger the deadlock scenario
						cancel()
						// Small delay to let cancellation propagate
						time.Sleep(10 * time.Millisecond)
					}
					return MarshalString(fmt.Sprintf("item-%d", i))
				},
			)
			done <- ret
		}()

		select {
		case ret := <-done:
			// Should return nil because remaining elements were skipped
			// after context cancellation caused semaphore Acquire to fail.
			assert.Nil(t, ret)
		case <-time.After(5 * time.Second):
			t.Fatal("deadlock detected: MarshalSliceConcurrently did not return within timeout")
		}
		cancel() // cleanup
	})

	t.Run("context already cancelled with worker limit", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		ctx, cancel := context.WithCancel(ctx)
		cancel() // Cancel before calling

		done := make(chan Array, 1)
		go func() {
			done <- MarshalSliceConcurrently(ctx, 10, 1, false, func(ctx context.Context, i int) Marshaler {
				return MarshalString("should not reach")
			})
		}()

		select {
		case ret := <-done:
			// Should return nil without deadlock since the context was
			// already cancelled and no elements could be marshaled.
			assert.Nil(t, ret)
		case <-time.After(5 * time.Second):
			t.Fatal("deadlock detected with pre-cancelled context")
		}
	})

	t.Run("cancelled context does not panic on MarshalGQL", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		ctx, cancel := context.WithCancel(ctx)
		cancel()

		ret := MarshalSliceConcurrently(
			ctx,
			3,
			1,
			false,
			func(ctx context.Context, i int) Marshaler {
				return MarshalString("ok")
			},
		)

		assert.Nil(t, ret)
		assert.NotPanics(t, func() {
			var buf bytes.Buffer
			ret.MarshalGQL(&buf)
		})
	})

	t.Run("no worker limit with cancelled context still works", func(t *testing.T) {
		// Without worker limit, there's no semaphore, so context cancellation
		// doesn't cause issues (all goroutines launch immediately).
		ctx := withTestResponseContext(context.Background())
		ctx, cancel := context.WithCancel(ctx)
		cancel()

		ret := MarshalSliceConcurrently(
			ctx,
			5,
			0,
			false,
			func(ctx context.Context, i int) Marshaler {
				return MarshalString("ok")
			},
		)
		require.Len(t, ret, 5)
	})
}

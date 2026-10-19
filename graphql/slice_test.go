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
		for i := range n {
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

	t.Run("omit panic handler does not recover", func(t *testing.T) {
		ctx := withTestResponseContext(context.Background())
		assert.Panics(t, func() {
			MarshalSliceConcurrently(ctx, 1, 0, true, func(ctx context.Context, i int) Marshaler {
				panic("test panic")
			})
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

// An element that is not marshalled, because its marshaller panicked or the
// context was cancelled before a worker could start it, holds Null: the same
// value an element whose resolver returned an error holds. A nullable element
// therefore reports null in place, and the generated non-null check nulls the
// list when elements are non-null. The rest of the list is kept.
func TestMarshalSliceConcurrentlyNullsElementsItCouldNotMarshal(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		length      int
		workerLimit int64
		cancelFirst bool
		marshal     func(cancel context.CancelFunc, i int) Marshaler
		wantJSON    string
		wantErrors  []string
	}{
		"single element panics": {
			length: 1,
			marshal: func(_ context.CancelFunc, _ int) Marshaler {
				panic("test panic")
			},
			wantJSON:   `[null]`,
			wantErrors: []string{"input: [0] internal system error"},
		},
		"one of several elements panics": {
			length: 3,
			marshal: func(_ context.CancelFunc, i int) Marshaler {
				if i == 1 {
					panic("test panic")
				}
				return MarshalString("ok")
			},
			wantJSON:   `["ok", null, "ok"]`,
			wantErrors: []string{"input: [1] internal system error"},
		},
		"context cancelled while a worker holds the only slot": {
			length:      5,
			workerLimit: 1,
			marshal: func(cancel context.CancelFunc, i int) Marshaler {
				// The worker still holds the semaphore, so the next Acquire
				// observes the cancellation rather than racing it.
				if i == 2 {
					cancel()
				}
				return MarshalString(fmt.Sprintf("item-%d", i))
			},
			wantJSON:   `["item-0", "item-1", "item-2", null, null]`,
			wantErrors: []string{"input: [3] context canceled"},
		},
		"context cancelled before the first worker": {
			length:      3,
			workerLimit: 1,
			cancelFirst: true,
			marshal: func(_ context.CancelFunc, _ int) Marshaler {
				return MarshalString("should not reach")
			},
			wantJSON:   `[null, null, null]`,
			wantErrors: []string{"input: [0] context canceled"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(withTestResponseContext(context.Background()))
			defer cancel()
			if tc.cancelFirst {
				cancel()
			}

			done := make(chan Array, 1)
			go func() {
				done <- MarshalSliceConcurrently(ctx, tc.length, tc.workerLimit, false,
					func(_ context.Context, i int) Marshaler { return tc.marshal(cancel, i) })
			}()

			var ret Array
			select {
			case ret = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("deadlock: MarshalSliceConcurrently did not return")
			}

			require.Len(t, ret, tc.length, "every element keeps its slot")
			var b bytes.Buffer
			ret.MarshalGQL(&b)
			assert.JSONEq(t, tc.wantJSON, b.String())

			var gotErrors []string
			for _, err := range GetErrors(ctx) {
				gotErrors = append(gotErrors, err.Message)
			}
			assert.Equal(t, tc.wantErrors, gotErrors)
		})
	}
}

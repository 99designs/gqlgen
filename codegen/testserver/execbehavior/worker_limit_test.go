package execbehavior

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"
)

func TestWorkerLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var running, peak atomic.Int32
		release := make(chan struct{})
		resolvers := &Stub{}
		resolvers.QueryResolver.LimitedItems = func(ctx context.Context, count int) ([]*LimitedItem, error) {
			items := make([]*LimitedItem, count)
			for i := range items {
				items[i] = &LimitedItem{ID: i}
			}
			return items, nil
		}
		resolvers.LimitedItemResolver.Slow = func(ctx context.Context, obj *LimitedItem) (int, error) {
			n := running.Add(1)
			defer running.Add(-1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			<-release
			return obj.ID * 10, nil
		}
		srv := newServer(resolvers, DirectiveRoot{})

		// Each element waits for a release, and is released only once every element that
		// the limit lets start has started.
		go func() {
			for range 8 {
				synctest.Wait()
				release <- struct{}{}
			}
		}()
		got := post(t, srv, `{ limitedItems(count: 8) { id slow } }`)
		items := make([]string, 8)
		for i := range items {
			items[i] = fmt.Sprintf(`{"id":%d,"slow":%d}`, i, i*10)
		}
		require.JSONEq(t, `{"data":{"limitedItems":[`+strings.Join(items, ",")+`]}}`, got)
		// The generated code passes worker_limit: 2 to the list marshaler, so no more than
		// two elements resolve at once, and with eight waiting elements two do.
		require.Equal(t, int32(2), peak.Load())
	})
}

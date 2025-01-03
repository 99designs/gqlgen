package graphql

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

type SchedulerFunc func(context.Context, *OperationContext, int, int) Scheduler

type Scheduler interface {
	Go(func(context.Context, int), int)
	Wait()
}

type defaultScheduler struct {
	ctx context.Context
	oc  *OperationContext
	n   int
	c   int
	wg  sync.WaitGroup
	sm  *semaphore.Weighted
}

func DefaultScheduler(ctx context.Context, oc *OperationContext, n, limit int) Scheduler {
	var sm *semaphore.Weighted
	if limit > 0 && n > limit {
		sm = semaphore.NewWeighted(int64(limit))
	}
	return &defaultScheduler{ctx: ctx, oc: oc, n: n, sm: sm}
}

func (g *defaultScheduler) Go(f func(context.Context, int), i int) {
	g.c++
	if g.c == g.n {
		// Run on-thread when either:
		// 1. There is only one concurrent task.
		// 2. This is last task that will run.
		f(g.ctx, i)
	} else if g.sm != nil {
		g.wg.Add(1)
		if err := g.sm.Acquire(g.ctx, 1); err != nil {
			defer g.wg.Done()
			g.oc.Error(g.ctx, err)
		} else {
			go func() {
				defer func() {
					g.sm.Release(1)
					g.wg.Done()
				}()
				f(g.ctx, i)
			}()
		}
	} else {
		g.wg.Add(1)
		go func() {
			defer g.wg.Done()
			f(g.ctx, i)
		}()
	}
}

func (g *defaultScheduler) Wait() {
	if g.n > 1 {
		g.wg.Wait()
	}
}

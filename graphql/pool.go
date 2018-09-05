package graphql

import (
	"sync/atomic"
)

// Pool is a pool of goroutines intended to allow easy reuse of goroutines and to
// optionally limit concurrency.
// When finished with the Pool, its Stop method must be called to avoid leaking goroutines.
// Pool is safe for concurrent use.
type Pool struct {
	curSize int64
	maxSize int64
	funcs   chan func()
}

// NewPool returns a new Pool that runs a maximum of maxSize goroutine at once.
// If maxSize <= 0, then the number of goroutines is unlimited.
func NewPool(maxSize int64) *Pool {
	return &Pool{
		maxSize: maxSize,
		funcs:   make(chan func()),
	}
}

// worker is the method that runs in all goroutines in the Pool.
// It receives off of the funcs channel, but also accepts parameter f to guarantee
// that that function will be executed first.
func (p *Pool) worker(f func()) {
	f()
	for f := range p.funcs {
		f()
	}
}

// Run runs f() in a pooled goroutine.
// If there is a goroutine limit and no goroutine is available,
// it blocks until one becomes available.
func (p *Pool) Go(f func()) {
	select {
	case p.funcs <- f:
	default:
		curSize := atomic.AddInt64(&p.curSize, 1)
		if p.maxSize <= 0 || curSize <= p.maxSize {
			go p.worker(f)
			return
		}
		// There's no goroutine available and we're at the limit. We must wait.
		p.funcs <- f
	}
}

// Stop stops all goroutines in the Pool.
func (p *Pool) Stop() {
	close(p.funcs)
}

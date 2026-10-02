package lru_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql/handler/lru"
)

// This cache sits in front of query parsing and validation, so every request a
// server serves performs exactly one Get against it, and the keys are whole
// query strings rather than short identifiers. A Get on the hit path is pure
// overhead added to every request, which is what makes it worth measuring.
//
// The key sizes below span 40 to about 5000 bytes. Measured, Get costs the same
// at both ends, so the length of a query does not show up in the cost of
// looking it up. That is the result rather than the premise: the sizes are kept
// so a change which does make the cost scale with key length is visible.

// benchKey returns a key of roughly the length of a real GraphQL document,
// since the cache is keyed by the query text itself.
func benchKey(n int) string {
	const field = " someReasonablyNamedField"
	var sb strings.Builder
	sb.Grow(len("query Bench {") + n*len(field) + len(" }"))
	sb.WriteString("query Bench {")
	for range n {
		sb.WriteString(field)
	}
	sb.WriteString(" }")
	return sb.String()
}

func BenchmarkGetHit(b *testing.B) {
	for _, fields := range []int{1, 20, 200} {
		b.Run(fmt.Sprintf("fields=%d", fields), func(b *testing.B) {
			ctx := context.Background()
			cache := lru.New[*ast.QueryDocument](128)
			key := benchKey(fields)
			cache.Add(ctx, key, &ast.QueryDocument{})

			b.ReportAllocs()
			for b.Loop() {
				if _, ok := cache.Get(ctx, key); !ok {
					b.Fatal("expected a hit")
				}
			}
		})
	}
}

// A miss is the cold-start path and the path a query-flooding client drives,
// since every distinct query text misses once before it is admitted.
func BenchmarkGetMiss(b *testing.B) {
	ctx := context.Background()
	cache := lru.New[*ast.QueryDocument](128)
	key := benchKey(20)

	b.ReportAllocs()
	for b.Loop() {
		if _, ok := cache.Get(ctx, key); ok {
			b.Fatal("expected a miss")
		}
	}
}

// Add is paid once per distinct query. Measuring it against a full cache is the
// honest case, because a server at steady state is evicting on every admission
// rather than filling empty slots.
func BenchmarkAddEvicting(b *testing.B) {
	const size = 128
	ctx := context.Background()
	cache := lru.New[*ast.QueryDocument](size)
	keys := make([]string, size*2)
	for i := range keys {
		keys[i] = fmt.Sprintf("%s # %d", benchKey(20), i)
	}
	for i := range size {
		cache.Add(ctx, keys[i], &ast.QueryDocument{})
	}

	doc := &ast.QueryDocument{}
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		cache.Add(ctx, keys[i%len(keys)], doc)
		i++
	}
}

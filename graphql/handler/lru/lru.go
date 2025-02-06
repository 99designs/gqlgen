package lru

import (
	"context"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/99designs/gqlgen/graphql"
)

type LRU[T any] struct {
	lru *lru.Cache[string, T]
}

var _ graphql.Cache[any] = &LRU[any]{}

func New[T any](size int) *LRU[T] {
	cache, err := lru.New[string, T](size)
	if err != nil {
		// An error is only returned for non-positive cache size
		// and we already checked for that.
		panic("unexpected error creating cache: " + err.Error())
	}
	return &LRU[T]{cache}
}

func (l LRU[T]) Get(ctx context.Context, key string) (value T, ok bool) {
	return l.lru.Get(key)
}

func (l LRU[T]) Add(ctx context.Context, key string, value T) {
	l.lru.Add(key, value)
}

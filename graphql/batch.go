package graphql

import (
	"context"
	"errors"
	"sync"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// BatchResult represents the result of a batch resolver for a single item.
type BatchResult[T any] struct {
	Value T
	Err   error
}

type batchContextKey struct{}

// BatchParentState holds the batch parent groups for the current context.
type BatchParentState struct {
	groups map[string]*BatchParentGroup
}

// BatchParentGroup represents a group of parent objects being resolved together.
type BatchParentGroup struct {
	Parents any
	fields  sync.Map
}

// BatchFieldResult represents the cached result of a batch field resolution.
type BatchFieldResult struct {
	once       sync.Once
	done       chan struct{}
	Results    any
	InvalidErr error
}

// WithBatchParents adds a batch parent group to the context.
func WithBatchParents(ctx context.Context, typeName string, parents any) context.Context {
	prev, _ := ctx.Value(batchContextKey{}).(*BatchParentState)
	var groups map[string]*BatchParentGroup
	if prev != nil {
		groups = make(map[string]*BatchParentGroup, len(prev.groups)+1)
		for k, v := range prev.groups {
			groups[k] = v
		}
	} else {
		groups = make(map[string]*BatchParentGroup, 1)
	}
	groups[typeName] = &BatchParentGroup{Parents: parents}

	return context.WithValue(ctx, batchContextKey{}, &BatchParentState{groups: groups})
}

// GetBatchParentGroup retrieves the batch parent group for a given type name from context.
func GetBatchParentGroup(ctx context.Context, typeName string) *BatchParentGroup {
	state, _ := ctx.Value(batchContextKey{}).(*BatchParentState)
	if state == nil {
		return nil
	}
	return state.groups[typeName]
}

// GetFieldResult retrieves or computes the result for a batch field.
func (g *BatchParentGroup) GetFieldResult(key string, resolve func() (any, error)) *BatchFieldResult {
	if g == nil {
		return nil
	}
	res, _ := g.fields.LoadOrStore(key, &BatchFieldResult{done: make(chan struct{})})
	result := res.(*BatchFieldResult)
	result.once.Do(func() {
		defer close(result.done)
		result.Results, result.InvalidErr = resolve()
	})
	<-result.done
	return result
}

// BatchParentIndex returns the index of the current parent in the batch from the path.
func BatchParentIndex(ctx context.Context) (ast.PathIndex, bool) {
	path := GetPath(ctx)
	if len(path) < 2 {
		return 0, false
	}
	if idx, ok := path[len(path)-2].(ast.PathIndex); ok {
		return idx, true
	}
	return 0, false
}

// BatchPathWithIndex returns a copy of the current path with the parent index replaced.
func BatchPathWithIndex(ctx context.Context, index int) ast.Path {
	path := GetPath(ctx)
	if len(path) < 2 {
		return path
	}
	if _, ok := path[len(path)-2].(ast.PathIndex); !ok {
		return path
	}
	copied := make(ast.Path, len(path))
	copy(copied, path)
	copied[len(path)-2] = ast.PathIndex(index)
	return copied
}

// AddBatchError adds an error for a specific index in a batch operation.
func AddBatchError(ctx context.Context, index int, err error) {
	if err == nil {
		return
	}
	path := BatchPathWithIndex(ctx, index)
	if list, ok := err.(gqlerror.List); ok {
		for _, item := range list {
			if item == nil {
				continue
			}
			if item.Path == nil {
				cloned := *item
				cloned.Path = path
				AddError(ctx, &cloned)
				continue
			}
			AddError(ctx, item)
		}
		return
	}
	var gqlErr *gqlerror.Error
	if errors.As(err, &gqlErr) {
		if gqlErr.Path == nil {
			cloned := *gqlErr
			cloned.Path = path
			AddError(ctx, &cloned)
			return
		}
		AddError(ctx, gqlErr)
		return
	}
	AddError(ctx, gqlerror.WrapPath(path, err))
}

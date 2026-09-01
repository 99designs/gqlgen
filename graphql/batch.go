package graphql

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// BatchErrors represents per-item errors from a batch resolver.
// The returned slice must be the same length as the results slice, with nils for successes.
type BatchErrors interface {
	error
	Errors() []error
}

// BatchErrorList is a simple BatchErrors implementation backed by a slice.
type BatchErrorList []error

func (e BatchErrorList) Error() string   { return "batch resolver returned errors" }
func (e BatchErrorList) Errors() []error { return []error(e) }
func (e BatchErrorList) Unwrap() []error {
	if len(e) == 0 {
		return nil
	}
	out := make([]error, 0, len(e))
	for _, err := range e {
		if err != nil {
			out = append(out, err)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type batchContextKey struct{}

// BatchParentState holds the batch parent groups for the current context.
type BatchParentState struct {
	groups map[string]*BatchParentGroup
}

// BatchParentGroup represents a group of parent objects being resolved together.
type BatchParentGroup struct {
	Parents any
	// IndexMap is used to remap original array indices to group-specific indices,
	// which is necessary when items are grouped by concrete type from an interface/union slice.
	IndexMap map[int]int
	fields   sync.Map
}

// BatchFieldResult represents the cached result of a batch field resolution.
type BatchFieldResult struct {
	once    sync.Once
	done    chan struct{}
	Results any
	Err     error
}

// WithBatchParents adds a batch parent group to the context.
func WithBatchParents(
	ctx context.Context,
	typeName string,
	parents any,
	indexMap map[int]int,
) context.Context {
	prev, _ := ctx.Value(batchContextKey{}).(*BatchParentState)
	var groups map[string]*BatchParentGroup
	if prev != nil {
		groups = make(map[string]*BatchParentGroup, len(prev.groups)+1)
		maps.Copy(groups, prev.groups)
	} else {
		groups = make(map[string]*BatchParentGroup, 1)
	}
	groups[typeName] = &BatchParentGroup{Parents: parents, IndexMap: indexMap}

	return context.WithValue(ctx, batchContextKey{}, &BatchParentState{groups: groups})
}

// WithBatchParentValues adds a batch parent group for a slice of values, taking the
// address of each element so that batch resolvers always receive a []*T.
//
// Generated marshalers hold parents as []*T when the model is nilable and as []T
// otherwise; batch resolvers only ever accept the pointer form. This bridges the two.
//
// The returned group aliases v: parents[i] is &v[i], not a copy. Callers must not
// mutate or grow v while the returned context is in use.
//
// Parents recovered from an interface or union slice are grouped by concrete type and
// need an index map; those callers build the pointer slice themselves and use
// [WithBatchParents] directly.
func WithBatchParentValues[T any](ctx context.Context, typeName string, v []T) context.Context {
	parents := make([]*T, len(v))
	for i := range v {
		parents[i] = &v[i]
	}
	return WithBatchParents(ctx, typeName, parents, nil)
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
func (g *BatchParentGroup) GetFieldResult(
	key string,
	resolve func() (any, error),
) *BatchFieldResult {
	if g == nil {
		return nil
	}
	res, _ := g.fields.LoadOrStore(key, &BatchFieldResult{done: make(chan struct{})})
	result := res.(*BatchFieldResult)
	result.once.Do(func() {
		defer close(result.done)
		result.Results, result.Err = resolve()
	})
	<-result.done
	return result
}

// BatchParents is the batch context for one field resolution: the sibling parents being
// resolved together, and the position within them of the parent this call is for.
type BatchParents[P any] struct {
	Parents  []P
	Index    ast.PathIndex
	IndexMap map[int]int

	group *BatchParentGroup
}

// BatchParentsFor returns the batch context registered for typeName.
//
// ok is false whenever the field is not being resolved as part of a batch: no group was
// registered, the group holds a different Go type, or the path carries no parent index.
// The caller must then resolve the single parent it was handed.
func BatchParentsFor[P any](ctx context.Context, typeName string) (BatchParents[P], bool) {
	group := GetBatchParentGroup(ctx, typeName)
	if group == nil {
		return BatchParents[P]{}, false
	}
	parents, ok := group.Parents.([]P)
	if !ok {
		return BatchParents[P]{}, false
	}
	idx, ok := BatchParentIndex(ctx)
	if !ok {
		return BatchParents[P]{}, false
	}
	return BatchParents[P]{
		Parents:  parents,
		Index:    idx,
		IndexMap: group.IndexMap,
		group:    group,
	}, true
}

// FieldResult resolves the field once for the whole group, memoized on the field's
// response key so that every sibling parent shares a single resolver call.
func (b BatchParents[P]) FieldResult(
	field CollectedField,
	resolve func() (any, error),
) *BatchFieldResult {
	key := field.Alias
	if key == "" {
		key = field.Name
	}
	return b.group.GetFieldResult(key, resolve)
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
	if gqlErr, ok := errors.AsType[*gqlerror.Error](err); ok {
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

// ResolveBatchGroupResult handles batch resolver results for grouped parents.
func ResolveBatchGroupResult[T any](
	ctx context.Context,
	idx ast.PathIndex,
	parentsLen int,
	result *BatchFieldResult,
	fieldName string,
	indexMap map[int]int,
) (any, error) {
	idxInt := int(idx)
	resultIdx := idxInt
	if indexMap != nil {
		mapped, ok := indexMap[idxInt]
		if !ok {
			panic(
				fmt.Sprintf(
					"batch resolver %s: index %d not found in index map (this is a gqlgen bug)",
					fieldName,
					idx,
				),
			)
		}
		resultIdx = mapped
	}
	if result.Err != nil {
		if batchErrs, ok := result.Err.(BatchErrors); ok {
			results, ok := result.Results.([]T)
			if !ok {
				AddBatchError(ctx, idxInt, fmt.Errorf(
					"batch resolver %s returned unexpected result type (index %d)",
					fieldName,
					idx,
				))
				return nil, nil
			}
			errs := batchErrs.Errors()
			if len(results) != parentsLen {
				AddBatchError(ctx, idxInt, fmt.Errorf(
					"index %d: batch resolver %s returned %d results for %d parents",
					idx,
					fieldName,
					len(results),
					parentsLen,
				))
				return nil, nil
			}
			if len(errs) != parentsLen {
				AddBatchError(ctx, idxInt, fmt.Errorf(
					"index %d: batch resolver %s returned %d errors for %d parents",
					idx,
					fieldName,
					len(errs),
					parentsLen,
				))
				return nil, nil
			}
			if resultIdx < 0 || resultIdx >= len(results) {
				AddBatchError(ctx, idxInt, fmt.Errorf(
					"batch resolver %s could not resolve parent index %d",
					fieldName,
					idx,
				))
				return nil, nil
			}
			if err := errs[resultIdx]; err != nil {
				AddBatchError(ctx, idxInt, err)
				return nil, nil
			}
			return results[resultIdx], nil
		}
		AddBatchError(ctx, idxInt, result.Err)
		return nil, nil
	}

	results, ok := result.Results.([]T)
	if !ok {
		AddBatchError(ctx, idxInt, fmt.Errorf(
			"batch resolver %s returned unexpected result type (index %d)",
			fieldName,
			idx,
		))
		return nil, nil
	}
	if len(results) != parentsLen {
		AddBatchError(ctx, idxInt, fmt.Errorf(
			"index %d: batch resolver %s returned %d results for %d parents",
			idx,
			fieldName,
			len(results),
			parentsLen,
		))
		return nil, nil
	}
	if resultIdx < 0 || resultIdx >= len(results) {
		AddBatchError(ctx, idxInt, fmt.Errorf(
			"batch resolver %s could not resolve parent index %d",
			fieldName,
			idx,
		))
		return nil, nil
	}
	return results[resultIdx], nil
}

// ResolveBatchSingleResult handles batch resolver results for a single parent.
func ResolveBatchSingleResult[T any](
	ctx context.Context,
	results []T,
	err error,
	fieldName string,
) (any, error) {
	if err != nil {
		if batchErrs, ok := err.(BatchErrors); ok {
			errs := batchErrs.Errors()
			if len(results) != 1 {
				AddBatchError(ctx, 0, fmt.Errorf(
					"batch resolver %s returned %d results for %d parents (index %d)",
					fieldName,
					len(results),
					1,
					0,
				))
				return nil, nil
			}
			if len(errs) != 1 {
				AddBatchError(ctx, 0, fmt.Errorf(
					"batch resolver %s returned %d errors for %d parents (index %d)",
					fieldName,
					len(errs),
					1,
					0,
				))
				return nil, nil
			}
			if errs[0] != nil {
				AddBatchError(ctx, 0, errs[0])
				return nil, nil
			}
			return results[0], nil
		}
		AddBatchError(ctx, 0, err)
		return nil, nil
	}
	if len(results) != 1 {
		AddBatchError(ctx, 0, fmt.Errorf(
			"batch resolver %s returned %d results for %d parents (index %d)",
			fieldName,
			len(results),
			1,
			0,
		))
		return nil, nil
	}
	return results[0], nil
}

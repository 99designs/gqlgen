package graphql

import (
	"context"
	"io"
	"iter"
	"slices"
	"sync"
	"sync/atomic"
)

// NewView creates a new view that observes a subset of m's fields. Callers
// must invoke NewView and AddIndices before m's first Dispatch: NewView
// appends to m.onResult without a lock and Dispatch iterates that slice from
// resolver goroutines, so adding a view concurrently with dispatch is a data
// race. Generated code respects this by constructing all views in a single
// goroutine before ProcessDeferredGroup spawns the dispatch goroutine.
func (m *FieldSet) NewView() *FieldSetView {
	view := &FieldSetView{fieldSet: m}

	m.onResult = append(m.onResult, func(ctx context.Context, fieldIndex int) {
		if !slices.Contains(view.indices, fieldIndex) {
			return
		}
		if view.pending.Add(-1) == 0 && view.onComplete != nil {
			view.onComplete(ctx)
		}
	})

	return view
}

type FieldSetView struct {
	indices    []int
	pending    atomic.Int32
	fieldSet   *FieldSet
	onComplete func(ctx context.Context)
}

// SetOnComplete sets a callback to be invoked when all the
// fields contained within the view have resolved.
//
// This method is NOT thread safe, and must not be invoked concurrently with any of its own methods,
// nor with any of the parent [FieldSet]'s methods.
func (v *FieldSetView) SetOnComplete(onComplete func(ctx context.Context)) {
	v.onComplete = onComplete
}

func (v *FieldSetView) AddIndices(indices ...int) {
	v.indices = append(v.indices, indices...)
	v.pending.Add(int32(len(indices)))
}

// MarshalGQL writes the JSON object containing only the fields the view's
// indices select. It must not be called before every index has been resolved —
// in normal use that means waiting for the onComplete callback registered via
// SetOnComplete.
func (v *FieldSetView) MarshalGQL(writer io.Writer) {
	marshalFieldSet(writer, v.consumeFieldValues())
}

// consumeFieldValues yields each (field, value) pair for the view's indices
// and nils the corresponding entries on the underlying [FieldSet] via
// [FieldSet.takeValues]. A second call from this view — or from any other
// view sharing the same index — yields nothing for the consumed indices.
func (v *FieldSetView) consumeFieldValues() iter.Seq2[*CollectedField, Marshaler] {
	return func(yield func(*CollectedField, Marshaler) bool) {
		values := v.fieldSet.takeValues(v.indices)
		for i, value := range values {
			if value == nil {
				continue
			}

			field := &v.fieldSet.fields[v.indices[i]]
			if !yield(field, value) {
				return
			}
		}
	}
}

type FieldSet struct {
	onResult    []func(ctx context.Context, fieldIndex int)
	fields      []CollectedField
	takeValueMu sync.Mutex
	Values      []Marshaler
	Invalids    uint32
	delayed     []delayedResult
}

type delayedResult struct {
	i int
	f func(context.Context) Marshaler
}

func NewFieldSet(fields []CollectedField) *FieldSet {
	return &FieldSet{
		fields: fields,
		Values: make([]Marshaler, len(fields)),
	}
}

// takeValues takes the values at the specified indices out of the [FieldSet.Values] slice,
// returning them in the same order as they were specified. If a [Marshaler] was previously
// taken at the specified index, the returned slice will hold a nil value at that index.
func (m *FieldSet) takeValues(indices []int) []Marshaler {
	result := make([]Marshaler, len(indices))
	m.takeValueMu.Lock()
	defer m.takeValueMu.Unlock()
	for i, valueI := range indices {
		if valueI >= len(m.Values) {
			continue
		}
		result[i] = m.Values[valueI]
		m.Values[valueI] = nil
	}

	return result
}

func (m *FieldSet) AddField(field CollectedField) {
	m.fields = append(m.fields, field)
	m.Values = append(m.Values, nil)
}

func (m *FieldSet) Concurrently(i int, f func(context.Context) Marshaler) {
	m.delayed = append(m.delayed, delayedResult{i: i, f: f})
}

func (m *FieldSet) Dispatch(ctx context.Context) {
	if len(m.delayed) == 1 {
		// only one concurrent task, no need to spawn a goroutine or deal create waitgroups
		m.executeDelayed(ctx, &m.delayed[0])
	} else if len(m.delayed) > 1 {
		// more than one concurrent task, use the main goroutine to do one, only spawn goroutines
		// for the others

		var wg sync.WaitGroup
		for _, d := range m.delayed[1:] {
			wg.Add(1)
			go func(d delayedResult) {
				defer wg.Done()
				m.executeDelayed(ctx, &d)
			}(d)
		}

		m.executeDelayed(ctx, &m.delayed[0])
		wg.Wait()
	}
}

func (m *FieldSet) executeDelayed(ctx context.Context, delayed *delayedResult) {
	result := delayed.f(ctx)
	m.Values[delayed.i] = result
	for _, fn := range m.onResult {
		fn(ctx, delayed.i)
	}
}

func (m *FieldSet) MarshalGQL(writer io.Writer) {
	marshalFieldSet(writer, m.allFieldValues())
}

func (m *FieldSet) allFieldValues() iter.Seq2[*CollectedField, Marshaler] {
	return func(yield func(*CollectedField, Marshaler) bool) {
		for i, field := range m.fields {
			if !yield(&field, m.Values[i]) {
				return
			}
		}
	}
}

func marshalFieldSet(writer io.Writer, fieldValues iter.Seq2[*CollectedField, Marshaler]) {
	writer.Write(openBrace)
	writtenFields := make(map[string]struct{})
	isFirst := true
	for field, marshaler := range fieldValues {
		if _, ok := writtenFields[field.Alias]; ok {
			continue
		}
		if !isFirst {
			writer.Write(comma)
		} else {
			isFirst = false
		}

		writeQuotedString(writer, field.Alias)
		writer.Write(colon)
		marshaler.MarshalGQL(writer)
		writtenFields[field.Alias] = struct{}{}
	}
	writer.Write(closeBrace)
}

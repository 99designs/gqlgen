package graphql

import (
	"context"
	"io"
	"iter"
	"slices"
	"sync"
)

func (m *FieldSet) NewView() *FieldSetView {
	view := &FieldSetView{
		fieldSet:       m,
		pendingResults: make(map[int]struct{}),
	}

	m.onResult = append(m.onResult, func(ctx context.Context, fieldIndex int) {
		view.pendingMu.RLock()
		pendingCount := len(view.pendingResults)
		view.pendingMu.RUnlock()
		if pendingCount == 0 {
			return
		}

		view.pendingMu.Lock()
		defer view.pendingMu.Unlock()
		if len(view.pendingResults) == 0 { // may have changed since we checked
			return
		}

		delete(view.pendingResults, fieldIndex)
		if len(view.pendingResults) == 0 && view.onComplete != nil {
			view.onComplete(ctx)
		}
	})

	return view
}

type FieldSetView struct {
	indices        []int
	pendingResults map[int]struct{}
	pendingMu      sync.RWMutex
	fieldSet       *FieldSet
	onComplete     func(ctx context.Context)
}

// SetOnComplete sets a callback to be invoked when all the
// fields contained within the view have resolved.
//
// This method is NOT thread safe, and must not be invoked concurrently with any of its own methods,
// nor with any of the parent [FieldSet]'s methods.
func (f *FieldSetView) SetOnComplete(onComplete func(ctx context.Context)) {
	f.onComplete = onComplete
}

func (f *FieldSetView) AddIndices(i ...int) *FieldSetView {
	f.indices = slices.Grow(f.indices, len(i))
	for _, v := range i {
		f.indices = append(f.indices, v)
		f.pendingResults[v] = struct{}{}
	}
	return f
}

// MarshalGQL should only be invoked after onFilled() has been invoked.
func (f *FieldSetView) MarshalGQL(writer io.Writer) {
	marshalFieldSet(writer, f.allFieldValues())
}

func (f *FieldSetView) allFieldValues() iter.Seq2[*CollectedField, Marshaler] {
	return func(yield func(*CollectedField, Marshaler) bool) {
		for i := range f.indices {
			field := &f.fieldSet.fields[i]
			value, ok := f.fieldSet.takeValue(i)
			if !ok {
				continue
			}

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

// takeValue takes the value at index i out of the [FieldSet.Values] slice
// and returns it, leaving a nil value in its place. The bool reflects whether a value was found at the specified index.
func (m *FieldSet) takeValue(i int) (Marshaler, bool) {
	m.takeValueMu.Lock()
	defer m.takeValueMu.Unlock()
	if i >= len(m.Values) {
		return nil, false
	}
	v := m.Values[i]
	m.Values[i] = nil
	return v, v != nil
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

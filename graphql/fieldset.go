package graphql

import (
	"context"
	"io"
)

type FieldSet struct {
	fields   []CollectedField
	Values   []Marshaler
	Invalids uint32
	delayed  []delayedResult
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

func (m *FieldSet) AddField(field CollectedField) {
	m.fields = append(m.fields, field)
	m.Values = append(m.Values, nil)
}

func (m *FieldSet) Concurrently(i int, f func(context.Context) Marshaler) {
	m.delayed = append(m.delayed, delayedResult{i: i, f: f})
}

func (m *FieldSet) Dispatch(ctx context.Context, oc *OperationContext) {
	sched := oc.Scheduler(ctx, len(m.delayed), 0)
	for _, d := range m.delayed {
		d := d
		f := func(gctx context.Context, i int) { m.Values[i] = d.f(gctx) }
		sched.Go(f, d.i)
	}
	sched.Wait()
}

func (m *FieldSet) MarshalGQL(writer io.Writer) {
	writer.Write(openBrace)
	for i, field := range m.fields {
		if i != 0 {
			writer.Write(comma)
		}
		writeQuotedString(writer, field.Alias)
		writer.Write(colon)
		m.Values[i].MarshalGQL(writer)
	}
	writer.Write(closeBrace)
}

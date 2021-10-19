package graphql

import (
	"context"
	"io"
)

var nullLit = []byte(`null`)
var trueLit = []byte(`true`)
var falseLit = []byte(`false`)
var openBrace = []byte(`{`)
var closeBrace = []byte(`}`)
var openBracket = []byte(`[`)
var closeBracket = []byte(`]`)
var colon = []byte(`:`)
var comma = []byte(`,`)

var Null = &lit{nullLit}
var True = &lit{trueLit}
var False = &lit{falseLit}

type Marshaler interface {
	MarshalGQL(w io.Writer)
}

type Unmarshaler interface {
	UnmarshalGQL(v interface{}) error
}

type ContextMarshaler interface {
	MarshalGQLContext(ctx context.Context, w io.Writer) error
}

type ContextUnmarshaler interface {
	UnmarshalGQLContext(ctx context.Context, v interface{}) error
}

type contextMarshalerAdapter struct {
	Context context.Context
	ContextMarshaler
}

func WrapContextMarshaler(ctx context.Context, m ContextMarshaler) Marshaler {
	return contextMarshalerAdapter{Context: ctx, ContextMarshaler: m}
}

func (a contextMarshalerAdapter) MarshalGQL(w io.Writer) {
	err := a.MarshalGQLContext(a.Context, w)
	if err != nil {
		AddError(a.Context, err)
		Null.MarshalGQL(w)
	}
}

type WriterFunc func(writer io.Writer)

func (f WriterFunc) MarshalGQL(w io.Writer) {
	f(w)
}

type ContextWriterFunc func(ctx context.Context, writer io.Writer) error

func (f ContextWriterFunc) MarshalGQLContext(ctx context.Context, w io.Writer) error {
	return f(ctx, w)
}

type Array []Marshaler

func (a Array) MarshalGQL(writer io.Writer) {
	writer.Write(openBracket)
	for i, val := range a {
		if i != 0 {
			writer.Write(comma)
		}
		val.MarshalGQL(writer)
	}
	writer.Write(closeBracket)
}

type lit struct{ b []byte }

func (l lit) MarshalGQL(w io.Writer) {
	w.Write(l.b)
}

func (l lit) MarshalGQLContext(ctx context.Context, w io.Writer) error {
	w.Write(l.b)
	return nil
}

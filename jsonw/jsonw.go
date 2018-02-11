package jsonw

import (
	"fmt"
	"io"
	"strconv"
	"time"
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

var Null = lit(nullLit)
var True = lit(trueLit)
var False = lit(falseLit)

type Writer interface {
	WriteJson(w io.Writer)
}

type OrderedMap struct {
	Keys   []string
	Values []Writer
}

type writerFunc func(writer io.Writer)

func (f writerFunc) WriteJson(w io.Writer) {
	f(w)
}

func NewOrderedMap(len int) *OrderedMap {
	return &OrderedMap{
		Keys:   make([]string, len),
		Values: make([]Writer, len),
	}
}

func (m *OrderedMap) Add(key string, value Writer) {
	m.Keys = append(m.Keys, key)
	m.Values = append(m.Values, value)
}

func (m *OrderedMap) WriteJson(writer io.Writer) {
	writer.Write(openBrace)
	for i, key := range m.Keys {
		if i != 0 {
			writer.Write(comma)
		}
		io.WriteString(writer, strconv.Quote(key))
		writer.Write(colon)
		m.Values[i].WriteJson(writer)
	}
	writer.Write(closeBrace)
}

type Array []Writer

func (a Array) WriteJson(writer io.Writer) {
	writer.Write(openBracket)
	for i, val := range a {
		if i != 0 {
			writer.Write(comma)
		}
		val.WriteJson(writer)
	}
	writer.Write(closeBracket)
}

func lit(b []byte) Writer {
	return writerFunc(func(w io.Writer) {
		w.Write(b)
	})
}

func Int(i int) Writer {
	return writerFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Itoa(i))
	})
}

func Float64(f float64) Writer {
	return writerFunc(func(w io.Writer) {
		io.WriteString(w, fmt.Sprintf("%f", f))
	})
}

func String(s string) Writer {
	return writerFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(s))
	})
}

func Bool(b bool) Writer {
	return writerFunc(func(w io.Writer) {
		if b {
			w.Write(trueLit)
		} else {
			w.Write(falseLit)
		}
	})
}

func Time(t time.Time) Writer {
	return writerFunc(func(w io.Writer) {
		io.WriteString(w, strconv.Quote(t.Format(time.RFC3339)))
	})
}

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

type Writer struct {
	out   io.Writer
	first bool
}

func New(w io.Writer) *Writer {
	return &Writer{
		out:   w,
		first: true,
	}
}

func (w *Writer) split() {
	if !w.first {
		w.out.Write(comma)
	}
	w.first = false
}

func (w *Writer) Null() {
	w.split()
	w.out.Write(nullLit)
}

func (w *Writer) Bool(v bool) {
	w.split()
	if v {
		w.out.Write(trueLit)
	} else {
		w.out.Write(falseLit)
	}
}

func (w *Writer) True() {
	w.split()
	w.out.Write(trueLit)
}

func (w *Writer) False() {
	w.split()
	w.out.Write(falseLit)
}

func (w *Writer) Int(v int) {
	w.split()
	io.WriteString(w.out, fmt.Sprintf("%d", v))
}

func (w *Writer) Float64(v float64) {
	w.split()
	io.WriteString(w.out, fmt.Sprintf("%f", v))
}

func (w *Writer) String(v string) {
	w.split()
	io.WriteString(w.out, strconv.Quote(v))
}

func (w *Writer) Time(t time.Time) {
	w.split()
	io.WriteString(w.out, strconv.Quote(t.Format(time.RFC3339)))
}

func (w *Writer) BeginObject() {
	w.split()

	w.first = true
	w.out.Write(openBrace)
}

func (w *Writer) ObjectKey(key string) {
	w.split()
	w.first = true

	io.WriteString(w.out, strconv.Quote(key))
	w.out.Write(colon)
}

func (w *Writer) EndObject() {
	w.first = false
	w.out.Write(closeBrace)
}

func (w *Writer) BeginArray() {
	w.split()

	w.first = true
	w.out.Write(openBracket)
}

func (w *Writer) EndArray() {
	w.first = false
	w.out.Write(closeBracket)
}

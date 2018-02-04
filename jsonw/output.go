package jsonw

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/vektah/graphql-go/errors"
)

var Null = literal{[]byte(`null`)}
var True = literal{[]byte(`true`)}
var False = literal{[]byte(`false`)}

var openParen = []byte(`{`)
var closeParen = []byte(`}`)
var openBracket = []byte(`[`)
var closeBracket = []byte(`]`)
var colon = []byte(`:`)
var comma = []byte(`,`)

type Encodable interface {
	JSON(w io.Writer)
}

type Map []struct {
	Name  string
	Value Encodable
}

func (r *Map) Set(name string, value Encodable) {
	*r = append(*r, struct {
		Name  string
		Value Encodable
	}{name, value})
}

func (r Map) JSON(w io.Writer) {
	w.Write(openParen)

	for i, f := range r {
		if i > 0 {
			w.Write(comma)
		}
		io.WriteString(w, strconv.Quote(f.Name))

		w.Write(colon)
		f.Value.JSON(w)
	}
	w.Write(closeParen)
}

type Response struct {
	Data       json.RawMessage        `json:"data,omitempty"`
	Errors     []*errors.QueryError   `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

type literal struct {
	value []byte
}

func (r literal) JSON(w io.Writer) {
	w.Write(r.value)
}

func Int(v int) Encodable {
	return literal{[]byte(fmt.Sprintf("%d", v))}
}

func ID(v int) Encodable {
	return literal{[]byte(fmt.Sprintf("%d", v))}
}

func String(v string) Encodable {
	return literal{[]byte(strconv.Quote(v))}
}

func Bool(v bool) Encodable {
	if v {
		return True
	} else {
		return False
	}
}

type Array []Encodable

func (r Array) JSON(w io.Writer) {
	w.Write(openBracket)

	for i, f := range r {
		if i > 0 {
			w.Write(comma)
		}

		f.JSON(w)
	}
	w.Write(closeBracket)
}

//go:generate go run ./inliner/inliner.go

package templates

import (
	"bytes"
	"strconv"
	"text/template"
	"unicode"

	"github.com/vektah/gqlgen/codegen"
)

func Run(e *codegen.Build) (*bytes.Buffer, error) {
	t := template.New("").Funcs(template.FuncMap{
		"ucFirst": ucFirst,
		"lcFirst": lcFirst,
		"quote":   strconv.Quote,
	})

	for filename, data := range data {
		_, err := t.New(filename).Parse(data)
		if err != nil {
			panic(err)
		}
	}

	buf := &bytes.Buffer{}
	err := t.Lookup("file.gotpl").Execute(buf, e)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func ucFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func lcFirst(s string) string {
	if s == "" {
		return ""
	}

	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

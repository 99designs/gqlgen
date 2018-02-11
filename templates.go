package main

import (
	"bytes"
	"path/filepath"
	"strconv"
	"text/template"
	"unicode"
)

func runTemplate(e *extractor) (*bytes.Buffer, error) {
	t, err := template.New("").Funcs(template.FuncMap{
		"ucFirst": ucFirst,
		"lcFirst": lcFirst,
		"quote":   strconv.Quote,
	}).ParseGlob(filepath.Join(templateDir, "templates", "*.gotpl"))
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	err = t.Lookup("gen").Execute(buf, e)
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

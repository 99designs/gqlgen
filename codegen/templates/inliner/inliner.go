package main

import (
	"bytes"
	"io/ioutil"
	"strconv"
	"strings"

	"golang.org/x/tools/imports"
)

func main() {
	dir := "./"

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	out := bytes.Buffer{}
	out.WriteString("package templates\n\n")
	out.WriteString("var data = map[string]string{\n")

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".gotpl") {
			continue
		}

		b, err := ioutil.ReadFile(dir + f.Name())
		if err != nil {
			panic(err)
		}

		out.WriteString(strconv.Quote(f.Name()))
		out.WriteRune(':')
		out.WriteString(strconv.Quote(string(b)))
		out.WriteString(",\n")
	}

	out.WriteString("}\n")

	formatted, err2 := imports.Process(dir+"data.go", out.Bytes(), nil)
	if err2 != nil {
		panic(err2)
	}

	ioutil.WriteFile(dir+"data.go", formatted, 0644)
}

package main

import (
	"bytes"
	"io/ioutil"

	"strconv"

	"golang.org/x/tools/imports"
)

func main() {
	out := bytes.Buffer{}
	out.WriteString("package introspection\n\n")
	out.WriteString("var Prelude = ")

	file, err := ioutil.ReadFile("prelude.graphql")
	if err != nil {
		panic(err)
	}

	out.WriteString(strconv.Quote(string(file)))
	out.WriteString("\n")

	formatted, err2 := imports.Process("prelude.go", out.Bytes(), nil)
	if err2 != nil {
		panic(err2)
	}

	ioutil.WriteFile("prelude.go", formatted, 0644)
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"syscall"

	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/imports"
)

var output = flag.String("out", "-", "the file to write to, - for stdout")
var schemaFilename = flag.String("schema", "schema.graphql", "the graphql schema to generate types from")
var typemap = flag.String("typemap", "types.json", "a json map going from graphql to golang types")
var packageName = flag.String("package", "", "the package name")
var help = flag.Bool("h", false, "this usage text")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s schema.graphql\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(1)
	}

	schema := schema.New()
	schemaRaw, err := ioutil.ReadFile(*schemaFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open schema: "+err.Error())
		os.Exit(1)
	}

	if err = schema.Parse(string(schemaRaw)); err != nil {
		fmt.Fprintln(os.Stderr, "unable to parse schema: "+err.Error())
		os.Exit(1)
	}

	e := extractor{
		PackageName: getPkgName(),
		goTypeMap:   loadTypeMap(),
		SchemaRaw:   string(schemaRaw),
		schema:      schema,
		Imports: map[string]string{
			"context": "context",
			"fmt":     "fmt",
			"io":      "io",
			"strconv": "strconv",
			"time":    "time",
			"reflect": "reflect",
			"strings": "strings",
			"sync":    "sync",

			"mapstructure":  "github.com/mitchellh/mapstructure",
			"introspection": "github.com/vektah/gqlgen/neelance/introspection",
			"errors":        "github.com/vektah/gqlgen/neelance/errors",
			"query":         "github.com/vektah/gqlgen/neelance/query",
			"schema":        "github.com/vektah/gqlgen/neelance/schema",
			"validation":    "github.com/vektah/gqlgen/neelance/validation",
			"jsonw":         "github.com/vektah/gqlgen/jsonw",
		},
	}
	e.extract()

	// Poke a few magic methods into query
	q := e.GetObject(e.QueryRoot)
	q.Fields = append(q.Fields, Field{
		Type:        e.getType("__Schema").Ptr(),
		GraphQLName: "__schema",
		NoErr:       true,
		MethodName:  "ec.introspectSchema",
		Object:      q,
	})
	q.Fields = append(q.Fields, Field{
		Type:        e.getType("__Type").Ptr(),
		GraphQLName: "__type",
		NoErr:       true,
		MethodName:  "ec.introspectType",
		Args:        []FieldArgument{{Name: "name", Type: kind{Scalar: true, Name: "string"}}},
		Object:      q,
	})

	if len(e.Errors) != 0 {
		for _, err := range e.Errors {
			fmt.Fprintln(os.Stderr, "err: "+err)
		}
		os.Exit(1)
	}

	if *output != "-" {
		_ = syscall.Unlink(*output)
	}

	if err = e.introspect(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	buf, err := runTemplate(&e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to generate code: "+err.Error())
		os.Exit(1)
	}

	if *output == "-" {
		fmt.Println(string(gofmt(*output, buf.Bytes())))
	} else {
		err := os.MkdirAll(filepath.Dir(*output), 0755)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to create directory: ", err.Error())
			os.Exit(1)
		}

		err = ioutil.WriteFile(*output, gofmt(*output, buf.Bytes()), 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to write output: ", err.Error())
			os.Exit(1)
		}
	}
}

func gofmt(filename string, b []byte) []byte {
	out, err := imports.Process(filename, b, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to gofmt: "+*output+":"+err.Error())
		return b
	}
	return out
}

func getPkgName() string {
	pkgName := *packageName
	if pkgName == "" {
		absPath, err := filepath.Abs(*output)
		if err != nil {
			panic(err)
		}
		pkgName = filepath.Base(filepath.Dir(absPath))
	}
	return pkgName
}

func loadTypeMap() map[string]string {
	goTypes := map[string]string{
		"__Directive":  "github.com/vektah/gqlgen/neelance/introspection.Directive",
		"__Type":       "github.com/vektah/gqlgen/neelance/introspection.Type",
		"__Field":      "github.com/vektah/gqlgen/neelance/introspection.Field",
		"__EnumValue":  "github.com/vektah/gqlgen/neelance/introspection.EnumValue",
		"__InputValue": "github.com/vektah/gqlgen/neelance/introspection.InputValue",
		"__Schema":     "github.com/vektah/gqlgen/neelance/introspection.Schema",
	}
	b, err := ioutil.ReadFile(*typemap)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open typemap: "+err.Error()+" creating it.")
		return goTypes
	}
	if err = json.Unmarshal(b, &goTypes); err != nil {
		fmt.Fprintln(os.Stderr, "unable to parse typemap: "+err.Error())
		os.Exit(1)
	}

	return goTypes
}

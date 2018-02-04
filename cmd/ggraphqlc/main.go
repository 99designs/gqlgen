package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"io"

	"github.com/vektah/graphql-go/schema"
)

var output = flag.String("out", "-", "the file to write to, - for stdout")
var schemaFilename = flag.String("schema", "schema.graphql", "the graphql schema to generate types from")
var typemap = flag.String("typemap", "types.json", "a json map going from graphql to golang types")
var packageName = flag.String("package", "graphql", "the package name")
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

	b, err := ioutil.ReadFile(*typemap)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open typemap: "+err.Error())
		os.Exit(1)
	}

	goTypes := map[string]string{
		"__Directive":  "github.com/vektah/graphql-go/introspection.Directive",
		"__Type":       "github.com/vektah/graphql-go/introspection.Type",
		"__Field":      "github.com/vektah/graphql-go/introspection.Field",
		"__EnumValue":  "github.com/vektah/graphql-go/introspection.EnumValue",
		"__InputValue": "github.com/vektah/graphql-go/introspection.InputValue",
		"__Schema":     "github.com/vektah/graphql-go/introspection.Schema",
		"Query":        "interface{}",
		"Mutation":     "interface{}",
	}
	if err = json.Unmarshal(b, &goTypes); err != nil {
		fmt.Fprintln(os.Stderr, "unable to parse typemap: "+err.Error())
		os.Exit(1)
	}

	e := extractor{
		PackageName: *packageName,
		goTypeMap:   goTypes,
		schemaRaw:   string(schemaRaw),
		Imports: map[string]string{
			"exec":   "github.com/vektah/graphql-go/exec",
			"jsonw":  "github.com/vektah/graphql-go/jsonw",
			"query":  "github.com/vektah/graphql-go/query",
			"schema": "github.com/vektah/graphql-go/schema",
		},
	}
	e.extract(schema)

	// Poke a few magic methods into query
	q := e.GetObject("Query")
	q.Fields = append(q.Fields, Field{
		Type:        e.getType("__Schema").Ptr(),
		GraphQLName: "__schema",
		NoErr:       true,
		MethodName:  "ec.IntrospectSchema",
	})
	q.Fields = append(q.Fields, Field{
		Type:        e.getType("__Type").Ptr(),
		GraphQLName: "__type",
		NoErr:       true,
		MethodName:  "ec.IntrospectType",
		Args:        []Arg{{Name: "name", Type: Type{Basic: true, Name: "string"}}},
	})

	if len(e.Errors) != 0 {
		for _, err := range e.Errors {
			fmt.Println(os.Stderr, "err: "+err)
		}
		os.Exit(1)
	}

	if err := e.introspect(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	var out io.Writer = os.Stdout
	if *output != "-" {
		outFile, err := os.Create(*output)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		defer outFile.Close()
		out = outFile
	}

	write(e, out)
}

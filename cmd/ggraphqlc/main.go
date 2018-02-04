package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/vektah/graphql-go/schema"
)

var output = flag.String("out", "gen.go", "the file to write to")
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

	goTypes := map[string]string{}
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

	outFile, err := os.Create(*output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	defer outFile.Close()
	write(e, outFile)
}

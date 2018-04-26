package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/vektah/gqlgen/codegen"
)

var output = flag.String("out", "generated.go", "the file to write to")
var models = flag.String("models", "models_gen.go", "the file to write the models to")
var schemaFilename = flag.String("schema", "schema.graphql", "the graphql schema to generate types from")
var typemap = flag.String("typemap", "", "a json map going from graphql to golang types")
var packageName = flag.String("package", "", "the package name")
var modelPackageName = flag.String("modelpackage", "", "the package name to use for models")
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

	schemaRaw, err := ioutil.ReadFile(*schemaFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open schema: "+err.Error())
		os.Exit(1)
	}

	types := loadTypeMap()

	err = codegen.Generate(codegen.Config{
		ModelFilename:    *models,
		ExecFilename:     *output,
		ExecPackageName:  *packageName,
		ModelPackageName: *modelPackageName,
		SchemaStr:        string(schemaRaw),
		Typemap:          types,
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
}

func loadTypeMap() map[string]string {
	var goTypes map[string]string
	if *typemap != "" {
		b, err := ioutil.ReadFile(*typemap)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to open typemap: "+err.Error())
			return nil
		}

		if err = json.Unmarshal(b, &goTypes); err != nil {
			fmt.Fprintln(os.Stderr, "unable to parse typemap: "+err.Error())
			os.Exit(1)
		}
	}

	return goTypes
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/vektah/gqlgen/codegen"
	"gopkg.in/yaml.v2"
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

func loadTypeMap() codegen.TypeMap {
	if *typemap == "" {
		return nil
	}

	if strings.HasSuffix(*typemap, ".json") {
		return loadTypeMapJSON()
	}

	return loadTypeMapYAML()
}

func loadTypeMapJSON() codegen.TypeMap {
	if *typemap == "" {
		return nil
	}

	b, err := ioutil.ReadFile(*typemap)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open typemap: "+err.Error())
		return nil
	}

	var typeMap codegen.TypeMap

	var goTypes map[string]codegen.TypeMapEntry
	if err = json.Unmarshal(b, &goTypes); err != nil {
		var oldGoTypes map[string]string
		if err = json.Unmarshal(b, &oldGoTypes); err != nil {
			fmt.Fprintln(os.Stderr, "unable to parse typemap: "+err.Error())
			os.Exit(1)
		}
		for typeName, entityPath := range oldGoTypes {
			typeMap = append(typeMap, codegen.TypeMapEntry{
				TypeName:   typeName,
				EntityPath: entityPath,
			})
		}

		return typeMap
	}
	for typeName, goType := range goTypes {
		goType.TypeName = typeName
		typeMap = append(typeMap, goType)
	}

	return typeMap
}

func loadTypeMapYAML() codegen.TypeMap {
	if *typemap == "" {
		return nil
	}

	b, err := ioutil.ReadFile(*typemap)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open typemap: "+err.Error())
		return nil
	}

	var typeMap codegen.TypeMap
	if err = yaml.Unmarshal(b, &typeMap); err != nil {
		fmt.Fprintln(os.Stderr, "unable to parse typemap: "+err.Error())
		os.Exit(1)
	}

	return typeMap
}

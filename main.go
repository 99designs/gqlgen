package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/vektah/gqlgen/codegen"
	"gopkg.in/yaml.v2"
)

var configFilename = flag.String("config", "", "the file to configuration to")
var output = flag.String("out", "", "the file to write to")
var models = flag.String("models", "", "the file to write the models to")
var schemaFilename = flag.String("schema", "", "the graphql schema to generate types from")
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

	var config *codegen.Config
	var err error
	if *configFilename != "" {
		config, err = codegen.LoadConfig(*configFilename)
	} else {
		config, err = codegen.LoadDefaultConfig()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	// overwrite by commandline options
	var emitYamlGuidance bool
	if *schemaFilename != "" {
		config.SchemaFilename = *schemaFilename
	}
	if *models != "" {
		config.Model.Filename = *models
	}
	if *output != "" {
		config.Exec.Filename = *output
	}
	if *packageName != "" {
		config.Exec.Package = *packageName
	}
	if *modelPackageName != "" {
		config.Model.Package = *modelPackageName
	}
	if *typemap != "" {
		config.Models = loadModelMap()
		emitYamlGuidance = true
	}

	schemaRaw, err := ioutil.ReadFile(config.SchemaFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open schema: "+err.Error())
		os.Exit(1)
	}
	config.SchemaStr = string(schemaRaw)

	if err = config.Check(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid config format: "+err.Error())
		os.Exit(1)
	}

	if emitYamlGuidance {
		var b []byte
		b, err = yaml.Marshal(config)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to marshal yaml: "+err.Error())
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "DEPRECATION WARNING: we are moving away from the json typemap, instead create a gqlgen.yml with the following content:\n\n%s\n", string(b))
	}

	err = codegen.Generate(*config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
}

func loadModelMap() codegen.TypeMap {
	var goTypes map[string]string
	b, err := ioutil.ReadFile(*typemap)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open typemap: "+err.Error())
		return nil
	}

	if err = yaml.Unmarshal(b, &goTypes); err != nil {
		fmt.Fprintln(os.Stderr, "unable to parse typemap: "+err.Error())
		os.Exit(1)
	}

	typeMap := make(codegen.TypeMap)
	for typeName, entityPath := range goTypes {
		typeMap[typeName] = codegen.TypeMapEntry{
			Model: entityPath,
		}
	}

	return typeMap
}

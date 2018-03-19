package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/vektah/gqlgen/codegen"
	"github.com/vektah/gqlgen/codegen/templates"
	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/imports"
)

var output = flag.String("out", "generated.go", "the file to write to")
var models = flag.String("models", "models_gen.go", "the file to write the models to")
var schemaFilename = flag.String("schema", "schema.graphql", "the graphql schema to generate types from")
var typemap = flag.String("typemap", "", "a json map going from graphql to golang types")
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

	_ = syscall.Unlink(*output)
	_ = syscall.Unlink(*models)

	types := loadTypeMap()

	modelsBuild := codegen.Models(schema, types, dirName())
	if len(modelsBuild.Models) > 0 {
		if *packageName != "" {
			modelsBuild.PackageName = *packageName
		}

		buf, err := templates.Run("models.gotpl", modelsBuild)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to generate code: "+err.Error())
			os.Exit(1)
		}

		write(*models, buf.Bytes())
		pkgName := fullPackageName()

		for _, model := range modelsBuild.Models {
			types[model.GQLType] = pkgName + "." + model.GoType
		}
	}

	build, err := codegen.Bind(schema, types, dirName())
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to generate code: "+err.Error())
		os.Exit(1)
	}
	build.SchemaRaw = string(schemaRaw)

	if *packageName != "" {
		build.PackageName = *packageName
	}

	buf, err := templates.Run("generated.gotpl", build)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to generate code: "+err.Error())
		os.Exit(1)
	}

	write(*output, buf.Bytes())
}

func gofmt(filename string, b []byte) []byte {
	out, err := imports.Process(filename, b, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to gofmt: "+err.Error())
		return b
	}
	return out
}

func write(filename string, b []byte) {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create directory: ", err.Error())
		os.Exit(1)
	}

	err = ioutil.WriteFile(filename, gofmt(filename, b), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %s", filename, err.Error())
		os.Exit(1)
	}
}

func absOutput() string {
	absPath, err := filepath.Abs(*output)
	if err != nil {
		panic(err)
	}
	return absPath
}

func dirName() string {
	return filepath.Dir(absOutput())
}

func fullPackageName() string {
	absPath, err := filepath.Abs(*output)
	if err != nil {
		panic(err)
	}
	pkgName := filepath.Dir(absPath)
	if *packageName != "" {
		pkgName = filepath.Join(filepath.Dir(pkgName), *packageName)
	}

	for _, gopath := range strings.Split(build.Default.GOPATH, ":") {
		gopath += "/src/"
		if strings.HasPrefix(pkgName, gopath) {
			pkgName = pkgName[len(gopath):]
		}
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
		"Int":          "github.com/vektah/gqlgen/graphql.Int",
		"Float":        "github.com/vektah/gqlgen/graphql.Float",
		"String":       "github.com/vektah/gqlgen/graphql.String",
		"Boolean":      "github.com/vektah/gqlgen/graphql.Boolean",
		"ID":           "github.com/vektah/gqlgen/graphql.ID",
		"Time":         "github.com/vektah/gqlgen/graphql.Time",
		"Map":          "github.com/vektah/gqlgen/graphql.Map",
	}
	if *typemap != "" {
		b, err := ioutil.ReadFile(*typemap)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to open typemap: "+err.Error())
			return goTypes
		}
		if err = json.Unmarshal(b, &goTypes); err != nil {
			fmt.Fprintln(os.Stderr, "unable to parse typemap: "+err.Error())
			os.Exit(1)
		}
	}

	return goTypes
}

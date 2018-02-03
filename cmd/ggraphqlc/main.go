package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/vektah/graphql-go/common"
	"github.com/vektah/graphql-go/schema"
)

var output = flag.String("out", "gen.go", "the file to write to")
var typemap = flag.String("typemap", "types.json", "a json map going from graphql to golang types")
var packageName = flag.String("package", "graphql", "the package name")
var help = flag.Bool("h", false, "this usage text")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s schema.graphql\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	args := flag.Args()

	if *help {
		flag.Usage()
		os.Exit(1)
	}

	if len(args) != 1 {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "\npath to schema is required")
		os.Exit(1)
	}

	schema := schema.New()
	b, err := ioutil.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open schema: "+err.Error())
		os.Exit(1)
	}

	if err = schema.Parse(string(b)); err != nil {
		fmt.Fprintln(os.Stderr, "unable to parse schema: "+err.Error())
		os.Exit(1)
	}

	b, err = ioutil.ReadFile(*typemap)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open typemap: "+err.Error())
		os.Exit(1)
	}

	goTypes := map[string]string{}
	if err = json.Unmarshal(b, &goTypes); err != nil {
		fmt.Fprintln(os.Stderr, "unable to parse typemap: "+err.Error())
		os.Exit(1)
	}

	vars := extract(schema, goTypes)
	if len(vars.Errors) != 0 {
		for _, err := range vars.Errors {
			fmt.Println(os.Stderr, err)
		}
		os.Exit(1)
	}

	for _, o := range vars.Objects {
		fmt.Println(o.Name)

		for _, f := range o.Fields {
			fmt.Println("  ", f.Name, f.ReturnType)
		}
	}
}

type goTypeMap map[string]string

func (m goTypeMap) get(name string) string {
	if fieldType, ok := m[name]; ok {
		return fieldType
	}
	fmt.Fprintf(os.Stderr, "unknown go type for %s, using interface{}. you should add it to types.json", name)
	return "interface{}"
}

func (m goTypeMap) buildGoTypeString(t common.Type) string {
	name := ""
	usePtr := true
	for {
		if _, nonNull := t.(*common.NonNull); nonNull {
			usePtr = false
		} else if _, nonNull := t.(*common.List); nonNull {
			usePtr = false
		} else {
			if usePtr {
				name += "*"
			}
			usePtr = true
		}

		switch val := t.(type) {
		case *common.NonNull:
			t = val.OfType
		case *common.List:
			name += "[]"
			t = val.OfType
		case *schema.Scalar:
			return val.Name
		case *schema.Object:
			return name + m.get(val.Name)
		case *common.TypeName:
			return name + m.get(val.Name)
		default:
			panic(fmt.Errorf("unknown type %T", t))
		}

	}
}

func extract(s *schema.Schema, goTypes goTypeMap) Vars {
	result := Vars{}
	for _, schemaType := range s.Types {

		switch schemaType := schemaType.(type) {
		case *schema.Object:
			if strings.HasPrefix(schemaType.Name, "__") {
				continue
			}
			object := Object{
				Name: schemaType.Name,
			}
			for _, field := range schemaType.Fields {
				object.Fields = append(object.Fields, Field{
					Name:       field.Name,
					ReturnType: goTypes.buildGoTypeString(field.Type),
				})
			}
			result.Objects = append(result.Objects, object)
		}
	}

	return result
}

type Vars struct {
	Errors  []string
	Objects []Object
}

type Object struct {
	Name   string
	Fields []Field
	goType string
}

type Field struct {
	Name       string
	ReturnType string
}

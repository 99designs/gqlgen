package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"path/filepath"
	"strconv"

	"bytes"

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

	e := extractor{
		goTypeMap: goTypes,
		imports:   map[string]string{},
	}
	e.extract(schema)

	if len(e.Errors) != 0 {
		for _, err := range e.Errors {
			fmt.Println(os.Stderr, "err: "+err)
		}
		os.Exit(1)
	}

	fmt.Println(e.String())
}

type extractor struct {
	Errors    []string
	Objects   []Object
	goTypeMap map[string]string
	imports   map[string]string // local -> full path
}

func (e *extractor) errorf(format string, args ...interface{}) {
	e.Errors = append(e.Errors, fmt.Sprintf(format, args...))
}

// get the type name to put in a file for a given fully resolved type, and add any imports required
// eg name = github.com/my/pkg.myType will return `pkg.myType` and add an import for `github.com/my/pkg`
func (e *extractor) get(name string) string {
	if fieldType, ok := e.goTypeMap[name]; ok {
		parts := strings.Split(fieldType, ".")
		if len(parts) == 1 {
			return parts[0]
		}

		packageName := strings.Join(parts[:len(parts)-1], ".")
		typeName := parts[len(parts)-1]

		localName := filepath.Base(packageName)
		i := 0
		for pkg, found := e.imports[localName]; found && pkg != packageName; localName = filepath.Base(packageName) + strconv.Itoa(i) {
			i++
			if i > 10 {
				panic("too many collisions")
			}
		}

		e.imports[localName] = packageName
		return localName + "." + typeName

	}
	fmt.Fprintf(os.Stderr, "unknown go type for %s, using interface{}. you should add it to types.json\n", name)
	return "interface{}"
}

func (e *extractor) buildGoTypeString(t common.Type) string {
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
			switch val.Name {
			case "String":
				return "string"
			case "Boolean":
				return "boolean"
			case "ID":
				return "string"
			case "Int":
				return "int"
			default:
				panic(fmt.Errorf("unknown scalar %s", val.Name))
			}
			return val.Name
		case *schema.Object:
			return name + e.get(val.Name)
		case *common.TypeName:
			return name + e.get(val.Name)
		default:
			panic(fmt.Errorf("unknown type %T", t))
		}

	}
}

func (e *extractor) extract(s *schema.Schema) {
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
				var args []Arg
				for _, arg := range field.Args {
					args = append(args, Arg{
						Name: arg.Name.Name,
						Type: e.buildGoTypeString(arg.Type),
					})
				}

				object.Fields = append(object.Fields, Field{
					Name:       field.Name,
					ReturnType: e.buildGoTypeString(field.Type),
					Args:       args,
				})
			}
			e.Objects = append(e.Objects, object)
		}
	}
}

func (e *extractor) String() string {
	b := &bytes.Buffer{}

	b.WriteString("imports:\n")
	for local, pkg := range e.imports {
		b.WriteString("\t" + local + " " + strconv.Quote(pkg) + "\n")
	}
	b.WriteString("\n")

	for _, o := range e.Objects {
		b.WriteString("object " + o.Name + ":\n")

		for _, f := range o.Fields {
			b.WriteString("\t" + f.Name + "(")

			first := true
			for _, arg := range f.Args {
				if !first {
					b.WriteString(", ")
				}
				first = false
				b.WriteString(arg.Name + " " + arg.Type)
			}

			b.WriteString(") " + f.ReturnType + "\n")
		}

		b.WriteString("\n")
	}

	return b.String()
}

type Object struct {
	Name   string
	Fields []Field
	goType string
}

type Field struct {
	Name       string
	ReturnType string
	Args       []Arg
}

type Arg struct {
	Name string
	Type string
}

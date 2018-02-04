package main

import (
	"bytes"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vektah/graphql-go/common"
	"github.com/vektah/graphql-go/schema"
	"golang.org/x/tools/go/loader"
)

type extractor struct {
	Errors      []string
	PackageName string
	Objects     []object
	goTypeMap   map[string]string
	Imports     map[string]string // local -> full path
	schemaRaw   string
}

func (e *extractor) errorf(format string, args ...interface{}) {
	e.Errors = append(e.Errors, fmt.Sprintf(format, args...))
}

// getType to put in a file for a given fully resolved type, and add any Imports required
// eg name = github.com/my/pkg.myType will return `pkg.myType` and add an import for `github.com/my/pkg`
func (e *extractor) getType(name string) Type {
	if fieldType, ok := e.goTypeMap[name]; ok {
		parts := strings.Split(fieldType, ".")
		if len(parts) == 1 {
			return Type{
				GraphQLName: name,
				Name:        parts[0],
			}
		}

		packageName := strings.Join(parts[:len(parts)-1], ".")
		typeName := parts[len(parts)-1]

		localName := filepath.Base(packageName)
		i := 0
		for pkg, found := e.Imports[localName]; found && pkg != packageName; localName = filepath.Base(packageName) + strconv.Itoa(i) {
			i++
			if i > 10 {
				panic("too many collisions")
			}
		}

		e.Imports[localName] = packageName
		return Type{
			GraphQLName: name,
			ImportedAs:  localName,
			Name:        typeName,
			Package:     packageName,
		}
	}
	fmt.Fprintf(os.Stderr, "unknown go type for %s, using interface{}. you should add it to types.json\n", name)
	return Type{
		GraphQLName: name,
		Name:        "interface{}",
	}
}

func (e *extractor) buildGoTypeString(t common.Type) Type {
	prefix := ""
	usePtr := true
	for {
		if _, nonNull := t.(*common.NonNull); nonNull {
			usePtr = false
		} else if _, nonNull := t.(*common.List); nonNull {
			usePtr = false
		} else {
			if usePtr {
				prefix += "*"
			}
			usePtr = true
		}

		switch val := t.(type) {
		case *common.NonNull:
			t = val.OfType
		case *common.List:
			prefix += "[]"
			t = val.OfType
		case *schema.Scalar:
			switch val.Name {
			case "String":
				return Type{Prefix: prefix, GraphQLName: "String", Name: "string"}
			case "Boolean":
				return Type{Prefix: prefix, GraphQLName: "Boolean", Name: "bool"}
			case "ID":
				return Type{Prefix: prefix, GraphQLName: "ID", Name: "int"}
			case "Int":
				return Type{Prefix: prefix, GraphQLName: "Int", Name: "int"}
			default:
				panic(fmt.Errorf("unknown scalar %s", val.Name))
			}
		case *schema.Object:
			t := e.getType(val.Name)
			t.Prefix = prefix
			return t
		case *common.TypeName:
			t := e.getType(val.Name)
			t.Prefix = prefix
			return t
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
			object := object{
				Name: schemaType.Name,
				Type: e.getType(schemaType.Name),
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
					Name: field.Name,
					Type: e.buildGoTypeString(field.Type),
					Args: args,
				})
			}
			e.Objects = append(e.Objects, object)
		}
	}
}

func (e *extractor) introspect() error {
	var conf loader.Config
	for _, name := range e.Imports {
		conf.Import(name)
	}

	prog, err := conf.Load()
	if err != nil {
		return err
	}

	for _, o := range e.Objects {
		if o.Type.Package == "" {
			continue
		}
		pkg := prog.Package(o.Type.Package)

		for ast, object := range pkg.Defs {
			if ast.Name != o.Type.Name {
				continue
			}

			e.findBindTargets(object.Type(), o)
		}
	}

	return nil
}

func (e *extractor) findBindTargets(t types.Type, object object) {
	switch t := t.(type) {
	case *types.Named:
		// Todo: bind to funcs?
		e.findBindTargets(t.Underlying(), object)

	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			field := t.Field(i)
			// Todo: struct tags, name and - at least

			// Todo: check for type matches before binding too?
			if objectField := object.GetField(field.Name()); objectField != nil {
				objectField.Bind = field.Name()
			}
		}
		t.Underlying()

	default:
		panic(fmt.Errorf("unknown type %T", t))
	}

}

func (e *extractor) String() string {
	b := &bytes.Buffer{}

	b.WriteString("Imports:\n")
	for local, pkg := range e.Imports {
		b.WriteString("\t" + local + " " + strconv.Quote(pkg) + "\n")
	}
	b.WriteString("\n")

	for _, o := range e.Objects {
		b.WriteString("object " + o.Name + ":\n")

		for _, f := range o.Fields {
			if f.Bind != "" {
				b.WriteString("\t" + f.Bind + " " + f.Type.Local() + "\n")
				continue
			}
			b.WriteString("\t" + o.Name + "_" + f.Name)

			b.WriteString("(")
			first := true
			for _, arg := range f.Args {
				if !first {
					b.WriteString(", ")
				}
				first = false
				b.WriteString(arg.Name + " " + arg.Type.Local())
			}
			b.WriteString(")")

			b.WriteString(" " + f.Type.Local() + "\n")
		}

		b.WriteString("\n")
	}

	return b.String()
}

type Type struct {
	GraphQLName string
	Name        string
	Package     string
	ImportedAs  string
	Prefix      string
}

func (t Type) Local() string {
	if t.ImportedAs == "" {
		return t.Prefix + t.Name
	}
	return t.Prefix + t.ImportedAs + "." + t.Name
}

type object struct {
	Name   string
	Fields []Field
	Type   Type
}

type Field struct {
	Name string
	Type Type
	Args []Arg
	Bind string
}

func (o *object) GetField(name string) *Field {
	for i, field := range o.Fields {
		if strings.EqualFold(field.Name, name) {
			return &o.Fields[i]
		}
	}
	return nil
}

type Arg struct {
	Name string
	Type Type
}

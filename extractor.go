package main

import (
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vektah/gqlgen/neelance/common"
	"github.com/vektah/gqlgen/neelance/schema"

	"go/build"

	"golang.org/x/tools/go/loader"
)

type extractor struct {
	Errors       []string
	PackageName  string
	Objects      []*object
	Interfaces   []*object
	goTypeMap    map[string]string
	Imports      map[string]string // local -> full path
	schema       *schema.Schema
	SchemaRaw    string
	QueryRoot    string
	MutationRoot string
}

func (e *extractor) extract() {
	for _, typ := range e.schema.Types {
		switch typ := typ.(type) {
		case *schema.Object:
			obj := &object{
				Name: typ.Name,
				Type: e.getType(typ.Name),
			}

			for _, i := range typ.Interfaces {
				obj.satisfies = append(obj.satisfies, i.Name)
			}

			for _, field := range typ.Fields {
				var args []FieldArgument
				for _, arg := range field.Args {
					args = append(args, FieldArgument{
						Name: arg.Name.Name,
						Type: e.buildType(arg.Type),
					})
				}

				obj.Fields = append(obj.Fields, Field{
					GraphQLName: field.Name,
					Type:        e.buildType(field.Type),
					Args:        args,
					Object:      obj,
				})
			}
			e.Objects = append(e.Objects, obj)
		case *schema.Union:
			obj := &object{
				Name: typ.Name,
				Type: e.buildType(typ),
			}
			e.Interfaces = append(e.Interfaces, obj)

		case *schema.Interface:
			obj := &object{
				Name: typ.Name,
				Type: e.buildType(typ),
			}
			e.Interfaces = append(e.Interfaces, obj)
		}

	}

	for name, typ := range e.schema.EntryPoints {
		obj := typ.(*schema.Object)
		e.GetObject(obj.Name).Root = true
		if name == "query" {
			e.QueryRoot = obj.Name
		}
		if name == "mutation" {
			e.MutationRoot = obj.Name
			e.GetObject(obj.Name).DisableConcurrency = true
		}
	}

	sort.Slice(e.Objects, func(i, j int) bool {
		return strings.Compare(e.Objects[i].Name, e.Objects[j].Name) == -1
	})

	sort.Slice(e.Interfaces, func(i, j int) bool {
		return strings.Compare(e.Interfaces[i].Name, e.Interfaces[j].Name) == -1
	})
}

func resolvePkg(pkgName string) (string, error) {
	cwd, _ := os.Getwd()

	pkg, err := build.Default.Import(pkgName, cwd, build.FindOnly)
	if err != nil {
		return "", err
	}

	return pkg.ImportPath, nil
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

		pkgName, err := resolvePkg(o.Type.Package)
		if err != nil {
			return fmt.Errorf("unable to resolve package: %s", o.Type.Package)
		}
		pkg := prog.Imported[pkgName]
		if pkg == nil {
			return fmt.Errorf("required package was not loaded: %s", pkgName)
		}

		for astNode, object := range pkg.Defs {
			if astNode.Name != o.Type.Name {
				continue
			}

			if e.findBindTargets(object.Type(), o) {
				break
			}
		}
	}

	return nil
}

func (e *extractor) errorf(format string, args ...interface{}) {
	e.Errors = append(e.Errors, fmt.Sprintf(format, args...))
}

func isOwnPkg(pkg string) bool {
	absPath, err := filepath.Abs(*output)
	if err != nil {
		panic(err)
	}

	return strings.HasSuffix(filepath.Dir(absPath), pkg)
}

// getType to put in a file for a given fully resolved type, and add any Imports required
// eg name = github.com/my/pkg.myType will return `pkg.myType` and add an import for `github.com/my/pkg`
func (e *extractor) getType(name string) kind {
	if fieldType, ok := e.goTypeMap[name]; ok {
		parts := strings.Split(fieldType, ".")
		if len(parts) == 1 {
			return kind{
				GraphQLName: name,
				Name:        parts[0],
			}
		}

		packageName := strings.Join(parts[:len(parts)-1], ".")
		typeName := parts[len(parts)-1]

		localName := ""
		if !isOwnPkg(packageName) {
			localName = filepath.Base(packageName)
			i := 0
			for pkg, found := e.Imports[localName]; found && pkg != packageName; localName = filepath.Base(packageName) + strconv.Itoa(i) {
				i++
				if i > 10 {
					panic("too many collisions")
				}
			}
		}
		e.Imports[localName] = packageName
		return kind{
			GraphQLName: name,
			ImportedAs:  localName,
			Name:        typeName,
			Package:     packageName,
		}
	}

	isRoot := false
	for _, s := range e.schema.EntryPoints {
		if s.(*schema.Object).Name == name {
			isRoot = true
			break
		}
	}

	if !isRoot {
		fmt.Fprintf(os.Stderr, "unknown go type for %s, using interface{}. you should add it to types.json\n", name)
	}
	e.goTypeMap[name] = "interface{}"
	return kind{
		GraphQLName: name,
		Name:        "interface{}",
	}
}

func (e *extractor) buildType(t common.Type) kind {
	var modifiers []string
	usePtr := true
	for {
		if _, nonNull := t.(*common.NonNull); nonNull {
			usePtr = false
		} else if _, nonNull := t.(*common.List); nonNull {
			usePtr = false
		} else {
			if usePtr {
				modifiers = append(modifiers, modPtr)
			}
			usePtr = true
		}

		switch val := t.(type) {
		case *common.NonNull:
			t = val.OfType
		case *common.List:
			modifiers = append(modifiers, modList)
			t = val.OfType
		case *schema.Scalar:
			var goType string

			switch val.Name {
			case "String":
				goType = "string"
			case "ID":
				goType = "string"
			case "Boolean":
				goType = "bool"
			case "Int":
				goType = "int"
			case "Float":
				goType = "float64"
			case "Time":
				return kind{
					Scalar:      true,
					Modifiers:   modifiers,
					GraphQLName: val.Name,
					Name:        "Time",
					Package:     "time",
					ImportedAs:  "time",
				}
			default:
				panic(fmt.Errorf("unknown scalar %s", val.Name))
			}
			return kind{
				Scalar:      true,
				Modifiers:   modifiers,
				GraphQLName: val.Name,
				Name:        goType,
			}
		case *schema.Object:
			t := e.getType(val.Name)
			t.Modifiers = modifiers
			return t
		case *common.TypeName:
			t := e.getType(val.Name)
			t.Modifiers = modifiers
			return t
		case *schema.Interface:
			t := e.getType(val.Name)
			t.Modifiers = modifiers
			if t.Modifiers[len(t.Modifiers)-1] == modPtr {
				t.Modifiers = t.Modifiers[0 : len(t.Modifiers)-1]
			}

			for _, implementor := range val.PossibleTypes {
				t.Implementors = append(t.Implementors, e.getType(implementor.Name))
			}

			return t
		case *schema.Union:
			t := e.getType(val.Name)
			t.Modifiers = modifiers
			if t.Modifiers[len(t.Modifiers)-1] == modPtr {
				t.Modifiers = t.Modifiers[0 : len(t.Modifiers)-1]
			}

			for _, implementor := range val.PossibleTypes {
				t.Implementors = append(t.Implementors, e.getType(implementor.Name))
			}

			return t
		case *schema.InputObject:
			t := e.getType(val.Name)
			t.Modifiers = modifiers
			return t
		case *schema.Enum:
			return kind{
				Scalar:      true,
				Modifiers:   modifiers,
				GraphQLName: val.Name,
				Name:        "string",
			}
		default:
			panic(fmt.Errorf("unknown type %T", t))
		}
	}
}

func (e *extractor) modifiersFromGoType(t types.Type) []string {
	var modifiers []string
	for {
		switch val := t.(type) {
		case *types.Pointer:
			modifiers = append(modifiers, modPtr)
			t = val.Elem()
		case *types.Array:
			modifiers = append(modifiers, modList)
			t = val.Elem()
		case *types.Slice:
			modifiers = append(modifiers, modList)
			t = val.Elem()
		default:
			return modifiers
		}
	}
}

func (e *extractor) findBindTargets(t types.Type, object *object) bool {
	switch t := t.(type) {
	case *types.Named:
		for i := 0; i < t.NumMethods(); i++ {
			method := t.Method(i)
			if !method.Exported() {
				continue
			}

			if methodField := object.GetField(method.Name()); methodField != nil {
				methodField.MethodName = "it." + method.Name()
				sig := method.Type().(*types.Signature)

				methodField.Type.Modifiers = e.modifiersFromGoType(sig.Results().At(0).Type())

				// check arg order matches code, not gql

				var newArgs []FieldArgument
			l2:
				for j := 0; j < sig.Params().Len(); j++ {
					param := sig.Params().At(j)
					for _, oldArg := range methodField.Args {
						if strings.EqualFold(oldArg.Name, param.Name()) {
							oldArg.Type.Modifiers = e.modifiersFromGoType(param.Type())
							newArgs = append(newArgs, oldArg)
							continue l2
						}
					}
					e.errorf("cannot match argument " + param.Name() + " to any argument in " + t.String())
				}
				methodField.Args = newArgs

				if sig.Results().Len() == 1 {
					methodField.NoErr = true
				} else if sig.Results().Len() != 2 {
					e.errorf("weird number of results on %s. expected either (result), or (result, error)", method.Name())
				}
			}
		}

		e.findBindTargets(t.Underlying(), object)
		return true

	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			field := t.Field(i)
			// Todo: struct tags, name and - at least

			if !field.Exported() {
				continue
			}

			// Todo: check for type matches before binding too?
			if objectField := object.GetField(field.Name()); objectField != nil {
				objectField.VarName = "it." + field.Name()
				objectField.Type.Modifiers = e.modifiersFromGoType(field.Type())
			}
		}
		t.Underlying()
		return true
	}

	return false
}

const (
	modList = "[]"
	modPtr  = "*"
)

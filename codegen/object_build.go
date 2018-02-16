package codegen

import (
	"fmt"
	"go/types"
	"os"
	"sort"
	"strings"

	"github.com/vektah/gqlgen/neelance/schema"
	"golang.org/x/tools/go/loader"
)

type Objects []*Object

func buildObjects(types NamedTypes, s *schema.Schema, prog *loader.Program) Objects {
	var objects Objects

	for _, typ := range s.Types {
		switch typ := typ.(type) {
		case *schema.Object:
			obj := buildObject(types, typ)

			if def := findGoType(prog, obj.Package, obj.GoType); def != nil {
				findBindTargets(def.Type(), obj)
			}

			objects = append(objects, obj)
		}
	}

	for name, typ := range s.EntryPoints {
		obj := typ.(*schema.Object)
		objects.ByName(obj.Name).Root = true
		if name == "mutation" {
			objects.ByName(obj.Name).DisableConcurrency = true
		}
	}

	sort.Slice(objects, func(i, j int) bool {
		return strings.Compare(objects[i].GQLType, objects[j].GQLType) == -1
	})

	return objects
}

func (os Objects) ByName(name string) *Object {
	for i, o := range os {
		if strings.EqualFold(o.GQLType, name) {
			return os[i]
		}
	}
	return nil
}

func buildObject(types NamedTypes, typ *schema.Object) *Object {
	obj := &Object{NamedType: types[typ.TypeName()]}

	for _, i := range typ.Interfaces {
		obj.Satisfies = append(obj.Satisfies, i.Name)
	}

	for _, field := range typ.Fields {
		var args []FieldArgument
		for _, arg := range field.Args {
			args = append(args, FieldArgument{
				GQLName: arg.Name.Name,
				Type:    types.getType(arg.Type),
			})
		}

		obj.Fields = append(obj.Fields, Field{
			GQLName: field.Name,
			Type:    types.getType(field.Type),
			Args:    args,
			Object:  obj,
		})
	}
	return obj
}

func findBindTargets(t types.Type, object *Object) bool {
	switch t := t.(type) {
	case *types.Named:
		for i := 0; i < t.NumMethods(); i++ {
			method := t.Method(i)
			if !method.Exported() {
				continue
			}

			if methodField := object.GetField(method.Name()); methodField != nil {
				methodField.GoMethodName = "it." + method.Name()
				sig := method.Type().(*types.Signature)

				methodField.Type.Modifiers = modifiersFromGoType(sig.Results().At(0).Type())

				// check arg order matches code, not gql

				var newArgs []FieldArgument
			l2:
				for j := 0; j < sig.Params().Len(); j++ {
					param := sig.Params().At(j)
					for _, oldArg := range methodField.Args {
						if strings.EqualFold(oldArg.GQLName, param.Name()) {
							oldArg.Type.Modifiers = modifiersFromGoType(param.Type())
							newArgs = append(newArgs, oldArg)
							continue l2
						}
					}
					fmt.Fprintln(os.Stderr, "cannot match argument "+param.Name()+" to any argument in "+t.String())
				}
				methodField.Args = newArgs

				if sig.Results().Len() == 1 {
					methodField.NoErr = true
				} else if sig.Results().Len() != 2 {
					fmt.Fprintf(os.Stderr, "weird number of results on %s. expected either (result), or (result, error)\n", method.Name())
				}
			}
		}

		findBindTargets(t.Underlying(), object)
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
				objectField.GoVarName = "it." + field.Name()
				objectField.Type.Modifiers = modifiersFromGoType(field.Type())
			}
		}
		t.Underlying()
		return true
	}

	return false
}

func modifiersFromGoType(t types.Type) []string {
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

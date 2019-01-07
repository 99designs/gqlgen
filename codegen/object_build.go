package codegen

import (
	"sort"

	"go/types"

	"log"

	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

func (g *Generator) buildObjects(ts NamedTypes, prog *loader.Program) (Objects, error) {
	var objects Objects

	for _, typ := range g.schema.Types {
		if typ.Kind != ast.Object {
			continue
		}

		obj, err := g.buildObject(ts, typ)
		if err != nil {
			return nil, err
		}

		if _, isMap := obj.GoType.(*types.Map); !isMap {
			for _, bindErr := range bindObject(obj, g.StructTag) {
				log.Println(bindErr.Error())
				log.Println("  Adding resolver method")
			}
		}

		objects = append(objects, obj)
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].GQLType < objects[j].GQLType
	})

	return objects, nil
}

var keywords = []string{
	"break",
	"default",
	"func",
	"interface",
	"select",
	"case",
	"defer",
	"go",
	"map",
	"struct",
	"chan",
	"else",
	"goto",
	"package",
	"switch",
	"const",
	"fallthrough",
	"if",
	"range",
	"type",
	"continue",
	"for",
	"import",
	"return",
	"var",
}

// sanitizeArgName prevents collisions with go keywords for arguments to resolver functions
func sanitizeArgName(name string) string {
	for _, k := range keywords {
		if name == k {
			return name + "Arg"
		}
	}
	return name
}

func (g *Generator) buildObject(ts NamedTypes, typ *ast.Definition) (*Object, error) {
	obj := &Object{TypeDefinition: ts[typ.Name]}
	typeEntry, entryExists := g.Models[typ.Name]

	tt := types.NewTypeName(0, g.Config.Exec.Pkg(), obj.GQLType+"Resolver", nil)
	obj.ResolverInterface = types.NewNamed(tt, nil, nil)

	if typ == g.schema.Query {
		obj.Root = true
	}

	if typ == g.schema.Mutation {
		obj.Root = true
		obj.DisableConcurrency = true
	}

	if typ == g.schema.Subscription {
		obj.Root = true
		obj.Stream = true
	}

	obj.Satisfies = append(obj.Satisfies, typ.Interfaces...)

	for _, intf := range g.schema.GetImplements(typ) {
		obj.Implements = append(obj.Implements, ts[intf.Name])
	}

	for _, field := range typ.Fields {
		if typ == g.schema.Query && field.Name == "__type" {
			obj.Fields = append(obj.Fields, Field{
				TypeReference:  &TypeReference{ts["__Schema"], []string{modPtr}, ast.NamedType("__Schema", nil)},
				GQLName:        "__schema",
				GoFieldType:    GoFieldMethod,
				GoReceiverName: "ec",
				GoFieldName:    "introspectSchema",
				Object:         obj,
				Description:    field.Description,
			})
			continue
		}
		if typ == g.schema.Query && field.Name == "__schema" {
			obj.Fields = append(obj.Fields, Field{
				TypeReference:  &TypeReference{ts["__Type"], []string{modPtr}, ast.NamedType("__Schema", nil)},
				GQLName:        "__type",
				GoFieldType:    GoFieldMethod,
				GoReceiverName: "ec",
				GoFieldName:    "introspectType",
				Args: []FieldArgument{
					{GQLName: "name", TypeReference: &TypeReference{ts["String"], []string{}, ast.NamedType("String", nil)}, Object: &Object{}},
				},
				Object: obj,
			})
			continue
		}

		var forceResolver bool
		var goName string
		if entryExists {
			if typeField, ok := typeEntry.Fields[field.Name]; ok {
				goName = typeField.FieldName
				forceResolver = typeField.Resolver
			}
		}

		var args []FieldArgument
		for _, arg := range field.Arguments {
			dirs, err := g.getDirectives(arg.Directives)
			if err != nil {
				return nil, err
			}
			newArg := FieldArgument{
				GQLName:       arg.Name,
				TypeReference: ts.getType(arg.Type),
				Object:        obj,
				GoVarName:     sanitizeArgName(arg.Name),
				Directives:    dirs,
			}

			if !newArg.TypeReference.IsInput && !newArg.TypeReference.IsScalar {
				return nil, errors.Errorf("%s cannot be used as argument of %s.%s. only input and scalar types are allowed", arg.Type, obj.GQLType, field.Name)
			}

			if arg.DefaultValue != nil {
				var err error
				newArg.Default, err = arg.DefaultValue.Value(nil)
				if err != nil {
					return nil, errors.Errorf("default value for %s.%s is not valid: %s", typ.Name, field.Name, err.Error())
				}
			}
			args = append(args, newArg)
		}

		obj.Fields = append(obj.Fields, Field{
			GQLName:       field.Name,
			TypeReference: ts.getType(field.Type),
			Args:          args,
			Object:        obj,
			GoFieldName:   goName,
			ForceResolver: forceResolver,
		})
	}

	return obj, nil
}

package unified

import (
	"go/types"
	"log"

	"strings"

	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

func (g *Schema) buildObject(typ *ast.Definition) (*Object, error) {
	obj := &Object{
		Definition: g.NamedTypes[typ.Name],
		InTypemap:  g.Config.Models.UserDefined(typ.Name),
	}

	tt := types.NewTypeName(0, g.Config.Exec.Pkg(), obj.Definition.GQLDefinition.Name+"Resolver", nil)
	obj.ResolverInterface = types.NewNamed(tt, nil, nil)

	if typ == g.Schema.Query {
		obj.Root = true
		obj.InTypemap = true
	}

	if typ == g.Schema.Mutation {
		obj.Root = true
		obj.DisableConcurrency = true
		obj.InTypemap = true
	}

	if typ == g.Schema.Subscription {
		obj.Root = true
		obj.Stream = true
		obj.InTypemap = true
	}

	obj.Satisfies = append(obj.Satisfies, typ.Interfaces...)

	for _, intf := range g.Schema.GetImplements(typ) {
		obj.Implements = append(obj.Implements, g.NamedTypes[intf.Name])
	}

	for _, field := range typ.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}

		f, err := g.buildField(obj, field)
		if err != nil {
			return nil, err
		}

		obj.Fields = append(obj.Fields, f)
	}

	dirs, err := g.getDirectives(typ.Directives)
	if err != nil {
		return nil, err
	}
	obj.Directives = dirs

	if _, isMap := obj.Definition.GoType.(*types.Map); !isMap && obj.InTypemap {
		for _, bindErr := range bindObject(obj, g.Config.StructTag) {
			log.Println(bindErr.Error())
			log.Println("  Adding resolver method")
		}
	}

	return obj, nil
}

func (g *Schema) buildField(obj *Object, field *ast.FieldDefinition) (*Field, error) {
	dirs, err := g.getDirectives(field.Directives)
	if err != nil {
		return nil, err
	}

	f := Field{
		GQLName:        field.Name,
		TypeReference:  g.NamedTypes.getType(field.Type),
		Object:         obj,
		Directives:     dirs,
		GoFieldName:    lintName(ucFirst(field.Name)),
		GoFieldType:    GoFieldVariable,
		GoReceiverName: "obj",
	}

	if field.DefaultValue != nil {
		var err error
		f.Default, err = field.DefaultValue.Value(nil)
		if err != nil {
			return nil, errors.Errorf("default value for %s.%s is not valid: %s", obj.Definition.GQLDefinition.Name, field.Name, err.Error())
		}
	}

	typeEntry, entryExists := g.Config.Models[obj.Definition.GQLDefinition.Name]
	if entryExists {
		if typeField, ok := typeEntry.Fields[field.Name]; ok {
			if typeField.Resolver {
				f.IsResolver = true
			}
			if typeField.FieldName != "" {
				f.GoFieldName = lintName(ucFirst(typeField.FieldName))
			}
		}
	}

	for _, arg := range field.Arguments {
		argDirs, err := g.getDirectives(arg.Directives)
		if err != nil {
			return nil, err
		}
		newArg := FieldArgument{
			GQLName:       arg.Name,
			TypeReference: g.NamedTypes.getType(arg.Type),
			Object:        obj,
			GoVarName:     sanitizeArgName(arg.Name),
			Directives:    argDirs,
		}

		if !newArg.TypeReference.Definition.GQLDefinition.IsInputType() {
			return nil, errors.Errorf("%s cannot be used as argument of %s.%s. only input and scalar types are allowed", arg.Type, obj.Definition.GQLDefinition.Name, field.Name)
		}

		if arg.DefaultValue != nil {
			var err error
			newArg.Default, err = arg.DefaultValue.Value(nil)
			if err != nil {
				return nil, errors.Errorf("default value for %s.%s is not valid: %s", obj.Definition.GQLDefinition.Name, field.Name, err.Error())
			}
		}
		f.Args = append(f.Args, newArg)
	}
	return &f, nil
}

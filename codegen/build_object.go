package codegen

import (
	"go/types"
	"log"
	"strings"

	"github.com/99designs/gqlgen/codegen/templates"

	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

func (b *builder) buildObject(typ *ast.Definition) (*Object, error) {
	dirs, err := b.getDirectives(typ.Directives)
	if err != nil {
		return nil, errors.Wrap(err, typ.Name)
	}

	isRoot := typ == b.Schema.Query || typ == b.Schema.Mutation || typ == b.Schema.Subscription

	obj := &Object{
		Definition:         b.NamedTypes[typ.Name],
		InTypemap:          b.Config.Models.UserDefined(typ.Name) || isRoot,
		Root:               isRoot,
		DisableConcurrency: typ == b.Schema.Mutation,
		Stream:             typ == b.Schema.Subscription,
		Directives:         dirs,
		ResolverInterface: types.NewNamed(
			types.NewTypeName(0, b.Config.Exec.Pkg(), typ.Name+"Resolver", nil),
			nil,
			nil,
		),
	}

	for _, intf := range b.Schema.GetImplements(typ) {
		obj.Implements = append(obj.Implements, b.NamedTypes[intf.Name])
	}

	for _, field := range typ.Fields {
		if strings.HasPrefix(field.Name, "__") {
			continue
		}

		f, err := b.buildField(obj, field)
		if err != nil {
			return nil, errors.Wrap(err, typ.Name+"."+field.Name)
		}

		if typ.Kind == ast.InputObject && !f.TypeReference.Definition.GQLDefinition.IsInputType() {
			return nil, errors.Errorf(
				"%s.%s: cannot use %s because %s is not a valid input type",
				typ.Name,
				field.Name,
				f.Definition.GQLDefinition.Name,
				f.TypeReference.Definition.GQLDefinition.Kind,
			)
		}

		obj.Fields = append(obj.Fields, f)
	}

	if obj.InTypemap && !isMap(obj.Definition.GoType) {
		for _, bindErr := range b.bindObject(obj) {
			log.Println(bindErr.Error())
			log.Println("  Adding resolver method")
		}
	}

	return obj, nil
}

func (b *builder) buildField(obj *Object, field *ast.FieldDefinition) (*Field, error) {
	dirs, err := b.getDirectives(field.Directives)
	if err != nil {
		return nil, err
	}

	f := Field{
		GQLName:        field.Name,
		TypeReference:  b.NamedTypes.getType(field.Type),
		Object:         obj,
		Directives:     dirs,
		GoFieldName:    templates.ToGo(field.Name),
		GoFieldType:    GoFieldVariable,
		GoReceiverName: "obj",
	}

	if field.DefaultValue != nil {
		var err error
		f.Default, err = field.DefaultValue.Value(nil)
		if err != nil {
			return nil, errors.Errorf("default value %s is not valid: %s", field.Name, err.Error())
		}
	}

	typeEntry, entryExists := b.Config.Models[obj.Definition.GQLDefinition.Name]
	if entryExists {
		if typeField, ok := typeEntry.Fields[field.Name]; ok {
			if typeField.Resolver {
				f.IsResolver = true
			}
			if typeField.FieldName != "" {
				f.GoFieldName = templates.ToGo(typeField.FieldName)
			}
		}
	}

	for _, arg := range field.Arguments {
		newArg, err := b.buildArg(obj, arg)
		if err != nil {
			return nil, err
		}
		f.Args = append(f.Args, newArg)
	}
	return &f, nil
}

func (b *builder) buildArg(obj *Object, arg *ast.ArgumentDefinition) (*FieldArgument, error) {
	argDirs, err := b.getDirectives(arg.Directives)
	if err != nil {
		return nil, err
	}
	newArg := FieldArgument{
		GQLName:       arg.Name,
		TypeReference: b.NamedTypes.getType(arg.Type),
		Object:        obj,
		GoVarName:     templates.ToGoPrivate(arg.Name),
		Directives:    argDirs,
	}

	if !newArg.TypeReference.Definition.GQLDefinition.IsInputType() {
		return nil, errors.Errorf(
			"cannot use %s as argument %s because %s is not a valid input type",
			newArg.Definition.GQLDefinition.Name,
			arg.Name,
			newArg.TypeReference.Definition.GQLDefinition.Kind,
		)
	}

	if arg.DefaultValue != nil {
		var err error
		newArg.Default, err = arg.DefaultValue.Value(nil)
		if err != nil {
			return nil, errors.Errorf("default value is not valid: %s", err.Error())
		}
	}

	return &newArg, nil
}

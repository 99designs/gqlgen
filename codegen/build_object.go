package codegen

import (
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

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

	typeEntry, entryExists := b.Config.Models[obj.Definition.Name]
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

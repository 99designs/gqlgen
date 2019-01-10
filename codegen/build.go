package codegen

import (
	"go/types"
	"sort"

	"fmt"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
	"golang.org/x/tools/go/loader"
)

type builder struct {
	Config     *config.Config
	Schema     *ast.Schema
	SchemaStr  map[string]string
	Program    *loader.Program
	Directives map[string]*Directive
	NamedTypes NamedTypes
}

func buildSchema(cfg *config.Config) (*Schema, error) {
	b := builder{
		Config: cfg,
	}

	var err error
	b.Schema, b.SchemaStr, err = cfg.LoadSchema()
	if err != nil {
		return nil, err
	}

	err = cfg.Check()
	if err != nil {
		return nil, err
	}

	progLoader := b.Config.NewLoaderWithoutErrors()
	b.Program, err = progLoader.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}

	b.NamedTypes = NamedTypes{}

	for _, schemaType := range b.Schema.Types {
		b.NamedTypes[schemaType.Name], err = b.buildTypeDef(schemaType)
		if err != nil {
			return nil, errors.Wrap(err, "unable to build type definition")
		}
	}

	b.Directives, err = b.buildDirectives()
	if err != nil {
		return nil, err
	}

	s := Schema{
		Config:     cfg,
		Directives: b.Directives,
		Schema:     b.Schema,
		SchemaStr:  b.SchemaStr,
	}

	for _, schemaType := range b.Schema.Types {
		switch schemaType.Kind {
		case ast.Object:
			obj, err := b.buildObject(schemaType)
			if err != nil {
				return nil, errors.Wrap(err, "unable to build object definition")
			}

			s.Objects = append(s.Objects, obj)
		case ast.InputObject:
			input, err := b.buildObject(schemaType)
			if err != nil {
				return nil, errors.Wrap(err, "unable to build input definition")
			}

			s.Inputs = append(s.Inputs, input)

		case ast.Union, ast.Interface:
			s.Interfaces = append(s.Interfaces, b.buildInterface(schemaType))

		case ast.Enum:
			if enum := b.buildEnum(schemaType); enum != nil {
				s.Enums = append(s.Enums, *enum)
			}
		}
	}

	if err := b.injectIntrospectionRoots(&s); err != nil {
		return nil, err
	}

	sort.Slice(s.Objects, func(i, j int) bool {
		return s.Objects[i].Definition.GQLDefinition.Name < s.Objects[j].Definition.GQLDefinition.Name
	})

	sort.Slice(s.Inputs, func(i, j int) bool {
		return s.Inputs[i].Definition.GQLDefinition.Name < s.Inputs[j].Definition.GQLDefinition.Name
	})

	sort.Slice(s.Interfaces, func(i, j int) bool {
		return s.Interfaces[i].Definition.GQLDefinition.Name < s.Interfaces[j].Definition.GQLDefinition.Name
	})

	sort.Slice(s.Enums, func(i, j int) bool {
		return s.Enums[i].Definition.GQLDefinition.Name < s.Enums[j].Definition.GQLDefinition.Name
	})

	return &s, nil
}

func (b *builder) injectIntrospectionRoots(s *Schema) error {
	obj := s.Objects.ByName(b.Schema.Query.Name)
	if obj == nil {
		return fmt.Errorf("root query type must be defined")
	}

	typeType, err := b.FindGoType("github.com/99designs/gqlgen/graphql/introspection", "Type")
	if err != nil {
		return errors.Wrap(err, "unable to find root Type introspection type")
	}

	obj.Fields = append(obj.Fields, &Field{
		TypeReference:  &TypeReference{b.NamedTypes["__Type"], types.NewPointer(typeType.Type()), ast.NamedType("__Schema", nil)},
		GQLName:        "__type",
		GoFieldType:    GoFieldMethod,
		GoReceiverName: "ec",
		GoFieldName:    "introspectType",
		Args: []*FieldArgument{
			{
				GQLName: "name",
				TypeReference: &TypeReference{
					b.NamedTypes["String"],
					types.Typ[types.String],
					ast.NamedType("String", nil),
				},
				Object: &Object{},
			},
		},
		Object: obj,
	})

	schemaType, err := b.FindGoType("github.com/99designs/gqlgen/graphql/introspection", "Schema")
	if err != nil {
		return errors.Wrap(err, "unable to find root Schema introspection type")
	}

	obj.Fields = append(obj.Fields, &Field{
		TypeReference:  &TypeReference{b.NamedTypes["__Schema"], types.NewPointer(schemaType.Type()), ast.NamedType("__Schema", nil)},
		GQLName:        "__schema",
		GoFieldType:    GoFieldMethod,
		GoReceiverName: "ec",
		GoFieldName:    "introspectSchema",
		Object:         obj,
	})

	return nil
}

func (b *builder) FindGoType(pkgName string, typeName string) (types.Object, error) {
	if pkgName == "" {
		return nil, nil
	}
	fullName := typeName
	if pkgName != "" {
		fullName = pkgName + "." + typeName
	}

	pkgName, err := resolvePkg(pkgName)
	if err != nil {
		return nil, errors.Errorf("unable to resolve package for %s: %s\n", fullName, err.Error())
	}

	pkg := b.Program.Imported[pkgName]
	if pkg == nil {
		return nil, errors.Errorf("required package was not loaded: %s", fullName)
	}

	for astNode, def := range pkg.Defs {
		if astNode.Name != typeName || def.Parent() == nil || def.Parent() != pkg.Pkg.Scope() {
			continue
		}

		return def, nil
	}

	return nil, errors.Errorf("unable to find type %s\n", fullName)
}

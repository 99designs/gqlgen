package unified

import (
	"go/types"
	"sort"

	"fmt"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/ast"
)

func NewSchema(cfg *config.Config) (*Schema, error) {
	g := Schema{
		Config: cfg,
	}

	var err error
	g.Schema, g.SchemaStr, err = cfg.LoadSchema()
	if err != nil {
		return nil, err
	}

	err = cfg.Check()
	if err != nil {
		return nil, err
	}

	progLoader := g.Config.NewLoaderWithoutErrors()
	g.Program, err = progLoader.Load()
	if err != nil {
		return nil, errors.Wrap(err, "loading failed")
	}

	g.NamedTypes = NamedTypes{}

	for _, schemaType := range g.Schema.Types {
		g.NamedTypes[schemaType.Name], err = g.buildTypeDef(schemaType)
		if err != nil {
			return nil, errors.Wrap(err, "unable to build type definition")
		}
	}

	g.Directives, err = g.buildDirectives()
	if err != nil {
		return nil, err
	}

	for _, schemaType := range g.Schema.Types {
		switch schemaType.Kind {
		case ast.Object:
			obj, err := g.buildObject(schemaType)
			if err != nil {
				return nil, errors.Wrap(err, "unable to build object definition")
			}

			g.Objects = append(g.Objects, obj)
		case ast.InputObject:
			input, err := g.buildInput(schemaType)
			if err != nil {
				return nil, errors.Wrap(err, "unable to build input definition")
			}

			g.Inputs = append(g.Inputs, input)

		case ast.Union, ast.Interface:
			g.Interfaces = append(g.Interfaces, g.buildInterface(schemaType))

		case ast.Enum:
			if enum := g.buildEnum(schemaType); enum != nil {
				g.Enums = append(g.Enums, *enum)
			}
		}
	}

	if err := g.injectIntrospectionRoots(); err != nil {
		return nil, err
	}

	sort.Slice(g.Objects, func(i, j int) bool {
		return g.Objects[i].Definition.GQLDefinition.Name < g.Objects[j].Definition.GQLDefinition.Name
	})

	sort.Slice(g.Inputs, func(i, j int) bool {
		return g.Inputs[i].Definition.GQLDefinition.Name < g.Inputs[j].Definition.GQLDefinition.Name
	})

	sort.Slice(g.Interfaces, func(i, j int) bool {
		return g.Interfaces[i].Definition.GQLDefinition.Name < g.Interfaces[j].Definition.GQLDefinition.Name
	})

	sort.Slice(g.Enums, func(i, j int) bool {
		return g.Enums[i].Definition.GQLDefinition.Name < g.Enums[j].Definition.GQLDefinition.Name
	})

	return &g, nil
}

func (g *Schema) injectIntrospectionRoots() error {
	obj := g.Objects.ByName(g.Schema.Query.Name)
	if obj == nil {
		return fmt.Errorf("root query type must be defined")
	}

	typeType, err := g.FindGoType("github.com/99designs/gqlgen/graphql/introspection", "Type")
	if err != nil {
		return errors.Wrap(err, "unable to find root Type introspection type")
	}

	obj.Fields = append(obj.Fields, &Field{
		TypeReference:  &TypeReference{g.NamedTypes["__Type"], types.NewPointer(typeType.Type()), ast.NamedType("__Schema", nil)},
		GQLName:        "__type",
		GoFieldType:    GoFieldMethod,
		GoReceiverName: "ec",
		GoFieldName:    "introspectType",
		Args: []FieldArgument{
			{
				GQLName: "name",
				TypeReference: &TypeReference{
					g.NamedTypes["String"],
					types.Typ[types.String],
					ast.NamedType("String", nil),
				},
				Object: &Object{},
			},
		},
		Object: obj,
	})

	schemaType, err := g.FindGoType("github.com/99designs/gqlgen/graphql/introspection", "Schema")
	if err != nil {
		return errors.Wrap(err, "unable to find root Schema introspection type")
	}

	obj.Fields = append(obj.Fields, &Field{
		TypeReference:  &TypeReference{g.NamedTypes["__Schema"], types.NewPointer(schemaType.Type()), ast.NamedType("__Schema", nil)},
		GQLName:        "__schema",
		GoFieldType:    GoFieldMethod,
		GoReceiverName: "ec",
		GoFieldName:    "introspectSchema",
		Object:         obj,
	})

	return nil
}

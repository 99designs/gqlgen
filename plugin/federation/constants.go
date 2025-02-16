package federation

import (
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
)

// The name of the field argument that is injected into the resolver to support @requires.
const fieldArgRequires = "_federationRequires"

// The name of the scalar type used in the injected field argument to support @requires.
const mapTypeName = "_RequiresMap"

// The @key directive that defines the key fields for an entity.
const dirNameKey = "key"

// The @requires directive that defines the required fields for an entity to be resolved.
const dirNameRequires = "requires"

// The @entityResolver directive allows users to specify entity resolvers as batch lookups
const dirNameEntityResolver = "entityResolver"

const dirNamePopulateFromRepresentations = "populateFromRepresentations"

var populateFromRepresentationsImplementation = `func(ctx context.Context, obj any, next graphql.Resolver) (res any, err error) {
	fc := graphql.GetFieldContext(ctx)

	// We get the Federation representations argument from the _entities resolver
	representations, ok := fc.Parent.Parent.Args["representations"].([]map[string]any)
	if !ok {
		return nil, errors.New("must be called from within _entities")
	}

	// Get the index of the current entity in the representations list. This is
	// set by the execution context after the _entities resolver is called.
	index := fc.Parent.Index
	if index == nil {
		return nil, errors.New("couldn't find input index for entity")
	}

	if len(representations) < *index {
		return nil, errors.New("representation not found")
	}

	return representations[*index], nil
}`

const DirNameEntityReference = "entityReference"

// The fields arguments must be provided to both key and requires directives.
const DirArgFields = "fields"

// Tells the code generator what type the directive is referencing
const DirArgType = "type"

// The file name for Federation directives
const dirGraphQLQFile = "federation/directives.graphql"

// The file name for Federation entities
const entityGraphQLQFile = "federation/entity.graphql"

const federationVersion1Schema = `
	directive @key(fields: _FieldSet!) repeatable on OBJECT | INTERFACE
	directive @requires(fields: _FieldSet!) on FIELD_DEFINITION
	directive @provides(fields: _FieldSet!) on FIELD_DEFINITION
	directive @extends on OBJECT | INTERFACE
	directive @external on FIELD_DEFINITION
	scalar _Any
	scalar _FieldSet
`

const federationVersion2Schema = `
	directive @authenticated on FIELD_DEFINITION | OBJECT | INTERFACE | SCALAR | ENUM
	directive @composeDirective(name: String!) repeatable on SCHEMA
	directive @extends on OBJECT | INTERFACE
	directive @external on OBJECT | FIELD_DEFINITION
	directive @key(fields: FieldSet!, resolvable: Boolean = true) repeatable on OBJECT | INTERFACE
	directive @inaccessible on
	  | ARGUMENT_DEFINITION
	  | ENUM
	  | ENUM_VALUE
	  | FIELD_DEFINITION
	  | INPUT_FIELD_DEFINITION
	  | INPUT_OBJECT
	  | INTERFACE
	  | OBJECT
	  | SCALAR
	  | UNION
	directive @interfaceObject on OBJECT
	directive @link(import: [String!], url: String!) repeatable on SCHEMA
	directive @override(from: String!, label: String) on FIELD_DEFINITION
	directive @policy(policies: [[federation__Policy!]!]!) on
	  | FIELD_DEFINITION
	  | OBJECT
	  | INTERFACE
	  | SCALAR
	  | ENUM
	directive @provides(fields: FieldSet!) on FIELD_DEFINITION
	directive @requires(fields: FieldSet!) on FIELD_DEFINITION
	directive @requiresScopes(scopes: [[federation__Scope!]!]!) on
	  | FIELD_DEFINITION
	  | OBJECT
	  | INTERFACE
	  | SCALAR
	  | ENUM
	directive @shareable repeatable on FIELD_DEFINITION | OBJECT
	directive @tag(name: String!) repeatable on
	  | ARGUMENT_DEFINITION
	  | ENUM
	  | ENUM_VALUE
	  | FIELD_DEFINITION
	  | INPUT_FIELD_DEFINITION
	  | INPUT_OBJECT
	  | INTERFACE
	  | OBJECT
	  | SCALAR
	  | UNION
	scalar _Any
	scalar FieldSet
	scalar federation__Policy
	scalar federation__Scope
`

var builtins = config.TypeMap{
	"_Service": {
		Model: config.StringList{
			"github.com/99designs/gqlgen/plugin/federation/fedruntime.Service",
		},
	},
	"_Entity": {
		Model: config.StringList{
			"github.com/99designs/gqlgen/plugin/federation/fedruntime.Entity",
		},
	},
	"Entity": {
		Model: config.StringList{
			"github.com/99designs/gqlgen/plugin/federation/fedruntime.Entity",
		},
	},
	"_Any": {
		Model: config.StringList{"github.com/99designs/gqlgen/graphql.Map"},
	},
	"federation__Scope": {
		Model: config.StringList{"github.com/99designs/gqlgen/graphql.String"},
	},
	"federation__Policy": {
		Model: config.StringList{"github.com/99designs/gqlgen/graphql.String"},
	},
}

var dirPopulateFromRepresentations = &ast.DirectiveDefinition{
	Name:         dirNamePopulateFromRepresentations,
	IsRepeatable: false,
	Description: `This is a runtime directive used to implement @requires. It's automatically placed
on the generated _federationRequires argument, and the implementation of it extracts the
correct value from the input representations list.`,
	Locations: []ast.DirectiveLocation{ast.LocationArgumentDefinition},
	Position: &ast.Position{Src: &ast.Source{
		Name: dirGraphQLQFile,
	}},
}

var dirEntityReference = &ast.DirectiveDefinition{
	Name:         DirNameEntityReference,
	IsRepeatable: false,
	Description: `This is a compile-time directive used to implement @requires.
It tells the code generator how to generate the model for the scalar.`,
	Locations: []ast.DirectiveLocation{ast.LocationScalar},
	Arguments: ast.ArgumentDefinitionList{
		{
			Name: DirArgType,
			Type: ast.NonNullNamedType("String", nil),
			Description: `The name of the entity that the fields selection
set should be validated against.`,
		},
		{
			Name:        DirArgFields,
			Type:        ast.NonNullNamedType("FieldSet", nil),
			Description: "The selection that the scalar should generate into.",
		},
	},
	Position: &ast.Position{Src: &ast.Source{
		Name: dirGraphQLQFile,
	}},
}

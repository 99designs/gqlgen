package exec

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/common"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

var metaSchema *schema.Schema
var schemaExec iExec
var typeExec iExec

func init() {
	metaSchema = schema.New()
	AddBuiltinScalars(metaSchema)
	if err := metaSchema.Parse(metaSchemaSrc); err != nil {
		panic(err)
	}

	{
		var err error
		schemaExec, err = makeWithType(metaSchema, metaSchema.Types["__Schema"], &schemaResolver{})
		if err != nil {
			panic(err)
		}
	}

	{
		var err error
		typeExec, err = makeWithType(metaSchema, metaSchema.Types["__Type"], &typeResolver{})
		if err != nil {
			panic(err)
		}
	}
}

func IntrospectSchema(s *schema.Schema) (interface{}, error) {
	r := &request{
		schema: s,
		doc:    introspectionQuery,
	}
	return introspectSchema(context.Background(), r, introspectionQuery.Operations["IntrospectionQuery"].SelSet), nil
}

func introspectSchema(ctx context.Context, r *request, selSet *query.SelectionSet) interface{} {
	return schemaExec.exec(ctx, r, selSet, reflect.ValueOf(&schemaResolver{r.schema}), false)
}

func introspectType(ctx context.Context, r *request, name string, selSet *query.SelectionSet) interface{} {
	t, ok := r.schema.Types[name]
	if !ok {
		return nil
	}
	return typeExec.exec(ctx, r, selSet, reflect.ValueOf(&typeResolver{t}), false)
}

var metaSchemaSrc = `
	# A Directive provides a way to describe alternate runtime execution and type validation behavior in a GraphQL document.
	#
	# In some cases, you need to provide options to alter GraphQL's execution behavior
	# in ways field arguments will not suffice, such as conditionally including or
	# skipping a field. Directives provide this by describing additional information
	# to the executor.
	type __Directive {
		name: String!
		description: String
		locations: [__DirectiveLocation!]!
		args: [__InputValue!]!
	}

	# A Directive can be adjacent to many parts of the GraphQL language, a
	# __DirectiveLocation describes one such possible adjacencies.
	enum __DirectiveLocation {
		# Location adjacent to a query operation.
		QUERY
		# Location adjacent to a mutation operation.
		MUTATION
		# Location adjacent to a subscription operation.
		SUBSCRIPTION
		# Location adjacent to a field.
		FIELD
		# Location adjacent to a fragment definition.
		FRAGMENT_DEFINITION
		# Location adjacent to a fragment spread.
		FRAGMENT_SPREAD
		# Location adjacent to an inline fragment.
		INLINE_FRAGMENT
		# Location adjacent to a schema definition.
		SCHEMA
		# Location adjacent to a scalar definition.
		SCALAR
		# Location adjacent to an object type definition.
		OBJECT
		# Location adjacent to a field definition.
		FIELD_DEFINITION
		# Location adjacent to an argument definition.
		ARGUMENT_DEFINITION
		# Location adjacent to an interface definition.
		INTERFACE
		# Location adjacent to a union definition.
		UNION
		# Location adjacent to an enum definition.
		ENUM
		# Location adjacent to an enum value definition.
		ENUM_VALUE
		# Location adjacent to an input object type definition.
		INPUT_OBJECT
		# Location adjacent to an input object field definition.
		INPUT_FIELD_DEFINITION
	}

	# One possible value for a given Enum. Enum values are unique values, not a
	# placeholder for a string or numeric value. However an Enum value is returned in
	# a JSON response as a string.
	type __EnumValue {
		name: String!
		description: String
		isDeprecated: Boolean!
		deprecationReason: String
	}

	# Object and Interface types are described by a list of Fields, each of which has
	# a name, potentially a list of arguments, and a return type.
	type __Field {
		name: String!
		description: String
		args: [__InputValue!]!
		type: __Type!
		isDeprecated: Boolean!
		deprecationReason: String
	}

	# Arguments provided to Fields or Directives and the input fields of an
	# InputObject are represented as Input Values which describe their type and
	# optionally a default value.
	type __InputValue {
		name: String!
		description: String
		type: __Type!
		# A GraphQL-formatted string representing the default value for this input value.
		defaultValue: String
	}

	# A GraphQL Schema defines the capabilities of a GraphQL server. It exposes all
	# available types and directives on the server, as well as the entry points for
	# query, mutation, and subscription operations.
	type __Schema {
		# A list of all types supported by this server.
		types: [__Type!]!
		# The type that query operations will be rooted at.
		queryType: __Type!
		# If this server supports mutation, the type that mutation operations will be rooted at.
		mutationType: __Type
		# If this server support subscription, the type that subscription operations will be rooted at.
		subscriptionType: __Type
		# A list of all directives supported by this server.
		directives: [__Directive!]!
	}

	# The fundamental unit of any GraphQL Schema is the type. There are many kinds of
	# types in GraphQL as represented by the ` + "`" + `__TypeKind` + "`" + ` enum.
	#
	# Depending on the kind of a type, certain fields describe information about that
	# type. Scalar types provide no information beyond a name and description, while
	# Enum types provide their values. Object and Interface types provide the fields
	# they describe. Abstract types, Union and Interface, provide the Object types
	# possible at runtime. List and NonNull types compose other types.
	type __Type {
		kind: __TypeKind!
		name: String
		description: String
		fields(includeDeprecated: Boolean = false): [__Field!]
		interfaces: [__Type!]
		possibleTypes: [__Type!]
		enumValues(includeDeprecated: Boolean = false): [__EnumValue!]
		inputFields: [__InputValue!]
		ofType: __Type
	}
	
	# An enum describing what kind of type a given ` + "`" + `__Type` + "`" + ` is.
	enum __TypeKind {
		# Indicates this type is a scalar.
		SCALAR
		# Indicates this type is an object. ` + "`" + `fields` + "`" + ` and ` + "`" + `interfaces` + "`" + ` are valid fields.
		OBJECT
		# Indicates this type is an interface. ` + "`" + `fields` + "`" + ` and ` + "`" + `possibleTypes` + "`" + ` are valid fields.
		INTERFACE
		# Indicates this type is a union. ` + "`" + `possibleTypes` + "`" + ` is a valid field.
		UNION
		# Indicates this type is an enum. ` + "`" + `enumValues` + "`" + ` is a valid field.
		ENUM
		# Indicates this type is an input object. ` + "`" + `inputFields` + "`" + ` is a valid field.
		INPUT_OBJECT
		# Indicates this type is a list. ` + "`" + `ofType` + "`" + ` is a valid field.
		LIST
		# Indicates this type is a non-null. ` + "`" + `ofType` + "`" + ` is a valid field.
		NON_NULL
	}
`

type schemaResolver struct {
	schema *schema.Schema
}

func (r *schemaResolver) Types() []*typeResolver {
	var l []*typeResolver
	addTypes := func(s *schema.Schema, metaOnly bool) {
		var names []string
		for name := range s.Types {
			if !metaOnly || strings.HasPrefix(name, "__") {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			l = append(l, &typeResolver{s.Types[name]})
		}
	}
	addTypes(r.schema, false)
	addTypes(metaSchema, true)
	return l
}

func (r *schemaResolver) QueryType() *typeResolver {
	t, ok := r.schema.EntryPoints["query"]
	if !ok {
		return nil
	}
	return &typeResolver{t}
}

func (r *schemaResolver) MutationType() *typeResolver {
	t, ok := r.schema.EntryPoints["mutation"]
	if !ok {
		return nil
	}
	return &typeResolver{t}
}

func (r *schemaResolver) SubscriptionType() *typeResolver {
	t, ok := r.schema.EntryPoints["subscription"]
	if !ok {
		return nil
	}
	return &typeResolver{t}
}

func (r *schemaResolver) Directives() []*directiveResolver {
	return nil
}

type typeResolver struct {
	typ common.Type
}

func (r *typeResolver) Kind() string {
	return r.typ.Kind()
}

func (r *typeResolver) Name() *string {
	if named, ok := r.typ.(schema.NamedType); ok {
		name := named.TypeName()
		return &name
	}
	return nil
}

func (r *typeResolver) Description() *string {
	return nil
}

func (r *typeResolver) Fields(args *struct{ IncludeDeprecated bool }) *[]*fieldResolver {
	var fields map[string]*schema.Field
	var fieldOrder []string
	switch t := r.typ.(type) {
	case *schema.Object:
		fields = t.Fields
		fieldOrder = t.FieldOrder
	case *schema.Interface:
		fields = t.Fields
		fieldOrder = t.FieldOrder
	default:
		return nil
	}

	l := make([]*fieldResolver, len(fieldOrder))
	for i, name := range fieldOrder {
		l[i] = &fieldResolver{fields[name]}
	}
	return &l
}

func (r *typeResolver) Interfaces() *[]*typeResolver {
	t, ok := r.typ.(*schema.Object)
	if !ok {
		return nil
	}

	l := make([]*typeResolver, len(t.Interfaces))
	for i, intf := range t.Interfaces {
		l[i] = &typeResolver{intf}
	}
	return &l
}

func (r *typeResolver) PossibleTypes() *[]*typeResolver {
	var possibleTypes []*schema.Object
	switch t := r.typ.(type) {
	case *schema.Interface:
		possibleTypes = t.PossibleTypes
	case *schema.Union:
		possibleTypes = t.PossibleTypes
	default:
		return nil
	}

	l := make([]*typeResolver, len(possibleTypes))
	for i, intf := range possibleTypes {
		l[i] = &typeResolver{intf}
	}
	return &l
}

func (r *typeResolver) EnumValues(args *struct{ IncludeDeprecated bool }) *[]*enumValueResolver {
	t, ok := r.typ.(*schema.Enum)
	if !ok {
		return nil
	}

	l := make([]*enumValueResolver, len(t.Values))
	for i, v := range t.Values {
		l[i] = &enumValueResolver{v}
	}
	return &l
}

func (r *typeResolver) InputFields() *[]*inputValueResolver {
	t, ok := r.typ.(*schema.InputObject)
	if !ok {
		return nil
	}

	l := make([]*inputValueResolver, len(t.FieldOrder))
	for i, name := range t.FieldOrder {
		l[i] = &inputValueResolver{t.Fields[name]}
	}
	return &l
}

func (r *typeResolver) OfType() *typeResolver {
	switch t := r.typ.(type) {
	case *common.List:
		return &typeResolver{t.OfType}
	case *common.NonNull:
		return &typeResolver{t.OfType}
	default:
		return nil
	}
}

type fieldResolver struct {
	field *schema.Field
}

func (r *fieldResolver) Name() string {
	return r.field.Name
}

func (r *fieldResolver) Description() *string {
	return nil
}

func (r *fieldResolver) Args() []*inputValueResolver {
	l := make([]*inputValueResolver, len(r.field.Args.FieldOrder))
	for i, name := range r.field.Args.FieldOrder {
		l[i] = &inputValueResolver{r.field.Args.Fields[name]}
	}
	return l
}

func (r *fieldResolver) Type() *typeResolver {
	return &typeResolver{r.field.Type}
}

func (r *fieldResolver) IsDeprecated() bool {
	return false
}

func (r *fieldResolver) DeprecationReason() *string {
	return nil
}

type inputValueResolver struct {
	value *common.InputValue
}

func (r *inputValueResolver) Name() string {
	return r.value.Name
}

func (r *inputValueResolver) Description() *string {
	return nil
}

func (r *inputValueResolver) Type() *typeResolver {
	return &typeResolver{r.value.Type}
}

func (r *inputValueResolver) DefaultValue() *string {
	if r.value.Default == nil {
		return nil
	}
	s := fmt.Sprint(r.value.Default)
	return &s
}

type enumValueResolver struct {
	value string
}

func (r *enumValueResolver) Name() string {
	return r.value
}

func (r *enumValueResolver) Description() *string {
	return nil
}

func (r *enumValueResolver) IsDeprecated() bool {
	return false
}

func (r *enumValueResolver) DeprecationReason() *string {
	return nil
}

type directiveResolver struct {
}

func (r *directiveResolver) Name() string {
	panic("TODO")
}

func (r *directiveResolver) Description() *string {
	panic("TODO")
}

func (r *directiveResolver) Locations() []string {
	panic("TODO")
}

func (r *directiveResolver) Args() []*inputValueResolver {
	panic("TODO")
}

var introspectionQuery *query.Document

func init() {
	var err *errors.QueryError
	introspectionQuery, err = query.Parse(introspectionQuerySrc, metaSchema.Resolve)
	if err != nil {
		panic(err)
	}
}

var introspectionQuerySrc = `
  query IntrospectionQuery {
    __schema {
      queryType { name }
      mutationType { name }
      subscriptionType { name }
      types {
        ...FullType
      }
      directives {
        name
        description
        locations
        args {
          ...InputValue
        }
      }
    }
  }
  fragment FullType on __Type {
    kind
    name
    description
    fields(includeDeprecated: true) {
      name
      description
      args {
        ...InputValue
      }
      type {
        ...TypeRef
      }
      isDeprecated
      deprecationReason
    }
    inputFields {
      ...InputValue
    }
    interfaces {
      ...TypeRef
    }
    enumValues(includeDeprecated: true) {
      name
      description
      isDeprecated
      deprecationReason
    }
    possibleTypes {
      ...TypeRef
    }
  }
  fragment InputValue on __InputValue {
    name
    description
    type { ...TypeRef }
    defaultValue
  }
  fragment TypeRef on __Type {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
                ofType {
                  kind
                  name
                }
              }
            }
          }
        }
      }
    }
  }
`

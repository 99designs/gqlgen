package exec

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

var metaSchema *schema.Schema
var schemaExec iExec
var typeExec iExec

func init() {
	{
		var err *errors.GraphQLError
		metaSchema, err = schema.Parse(metaSchemaSrc)
		if err != nil {
			panic(err)
		}
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

func introspectSchema(r *request, selSet *query.SelectionSet) interface{} {
	return schemaExec.exec(r, selSet, reflect.ValueOf(&schemaResolver{r.schema}))
}

func introspectType(r *request, name string, selSet *query.SelectionSet) interface{} {
	t, ok := r.schema.Types[name]
	if !ok {
		return nil
	}
	return typeExec.exec(r, selSet, reflect.ValueOf(&typeResolver{typ: t}))
}

var metaSchemaSrc = `
	type __Schema {
		types: [__Type!]!
		queryType: __Type!
		mutationType: __Type
		directives: [__Directive!]!
	}

	type __Type {
		kind: __TypeKind!
		name: String
		description: String

		# OBJECT and INTERFACE only
		fields(includeDeprecated: Boolean = false): [__Field!]

		# OBJECT only
		interfaces: [__Type!]

		# INTERFACE and UNION only
		possibleTypes: [__Type!]

		# ENUM only
		enumValues(includeDeprecated: Boolean = false): [__EnumValue!]

		# INPUT_OBJECT only
		inputFields: [__InputValue!]

		# NON_NULL and LIST only
		ofType: __Type
	}

	type __Field {
		name: String!
		description: String
		args: [__InputValue!]!
		type: __Type!
		isDeprecated: Boolean!
		deprecationReason: String
	}

	type __InputValue {
		name: String!
		description: String
		type: __Type!
		defaultValue: String
	}

	type __EnumValue {
		name: String!
		description: String
		isDeprecated: Boolean!
		deprecationReason: String
	}

	enum __TypeKind {
		SCALAR
		OBJECT
		INTERFACE
		UNION
		ENUM
		INPUT_OBJECT
		LIST
		NON_NULL
	}

	type __Directive {
		name: String!
		description: String
		locations: [__DirectiveLocation!]!
		args: [__InputValue!]!
	}

	enum __DirectiveLocation {
		QUERY
		MUTATION
		FIELD
		FRAGMENT_DEFINITION
		FRAGMENT_SPREAD
		INLINE_FRAGMENT
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
	return &typeResolver{typ: r.schema.Types[r.schema.EntryPoints["query"]]}
}

func (r *schemaResolver) MutationType() *typeResolver {
	return &typeResolver{typ: r.schema.Types[r.schema.EntryPoints["mutation"]]}
}

func (r *schemaResolver) Directives() []*directiveResolver {
	return nil
}

type typeResolver struct {
	typ schema.Type
}

func (r *typeResolver) Kind() string {
	switch r.typ.(type) {
	case *schema.Scalar:
		return "SCALAR"
	case *schema.Object:
		return "OBJECT"
	case *schema.Interface:
		return "INTERFACE"
	case *schema.Union:
		return "UNION"
	case *schema.Enum:
		return "ENUM"
	case *schema.InputObject:
		return "INPUT_OBJECT"
	case *schema.List:
		return "LIST"
	case *schema.NonNull:
		return "NON_NULL"
	default:
		panic("unreachable")
	}
}

func (r *typeResolver) Name() *string {
	switch t := r.typ.(type) {
	case *schema.Scalar:
		return &t.Name
	case *schema.Object:
		return &t.Name
	case *schema.Interface:
		return &t.Name
	case *schema.Union:
		return &t.Name
	case *schema.Enum:
		return &t.Name
	case *schema.InputObject:
		return &t.Name
	default:
		return nil
	}
}

func (r *typeResolver) Description() string {
	return ""
}

func (r *typeResolver) Fields(args struct{ IncludeDeprecated bool }) *[]*fieldResolver {
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

func (r *typeResolver) EnumValues(args struct{ IncludeDeprecated bool }) *[]*enumValueResolver {
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
	panic("TODO")
}

func (r *typeResolver) OfType() *typeResolver {
	panic("TODO")
}

type fieldResolver struct {
	field *schema.Field
}

func (r *fieldResolver) Name() string {
	return r.field.Name
}

func (r *fieldResolver) Description() string {
	return ""
}

func (r *fieldResolver) Args() []*inputValueResolver {
	l := make([]*inputValueResolver, len(r.field.ArgOrder))
	for i, name := range r.field.ArgOrder {
		l[i] = &inputValueResolver{r.field.Args[name]}
	}
	return l
}

func (r *fieldResolver) Type() *typeResolver {
	return &typeResolver{typ: r.field.Type}
}

func (r *fieldResolver) IsDeprecated() bool {
	return false
}

func (r *fieldResolver) DeprecationReason() *string {
	return nil
}

type inputValueResolver struct {
	value *schema.InputValue
}

func (r *inputValueResolver) Name() string {
	return r.value.Name
}

func (r *inputValueResolver) Description() string {
	return ""
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

func (r *enumValueResolver) Description() string {
	return ""
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

func (r *directiveResolver) Description() string {
	panic("TODO")
}

func (r *directiveResolver) Locations() []string {
	panic("TODO")
}

func (r *directiveResolver) Args() []*inputValueResolver {
	panic("TODO")
}

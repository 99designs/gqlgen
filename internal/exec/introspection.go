package exec

import (
	"reflect"
	"sort"

	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

var metaSchema *schema.Schema
var schemaExec iExec
var typeExec iExec

func init() {
	var err error
	metaSchema, err = schema.Parse(metaSchemaSrc, "")
	if err != nil {
		panic(err)
	}

	schemaExec, err = makeExec(metaSchema, metaSchema.AllTypes["__Schema"], reflect.TypeOf(&schemaResolver{}), make(map[typeRefMapKey]*typeRefExec))
	if err != nil {
		panic(err)
	}

	typeExec, err = makeExec(metaSchema, metaSchema.AllTypes["__Type"], reflect.TypeOf(&typeResolver{}), make(map[typeRefMapKey]*typeRefExec))
	if err != nil {
		panic(err)
	}
}

func introspectSchema(r *request, selSet *query.SelectionSet) interface{} {
	return schemaExec.exec(r, selSet, reflect.ValueOf(&schemaResolver{r.Schema}))
}

func introspectType(r *request, name string, selSet *query.SelectionSet) interface{} {
	t, ok := r.Schema.AllTypes[name]
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
	addTypes := func(s *schema.Schema) {
		var names []string
		for name := range s.AllTypes {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			l = append(l, &typeResolver{typ: s.AllTypes[name]})
		}
	}
	addTypes(r.schema)
	addTypes(metaSchema)
	for _, name := range scalarTypeNames {
		l = append(l, &typeResolver{scalar: name})
	}
	return l
}

func (r *schemaResolver) QueryType() *typeResolver {
	return &typeResolver{typ: r.schema.AllTypes[r.schema.EntryPoints["query"]]}
}

func (r *schemaResolver) MutationType() *typeResolver {
	return &typeResolver{typ: r.schema.AllTypes[r.schema.EntryPoints["mutation"]]}
}

func (r *schemaResolver) Directives() []*directiveResolver {
	panic("TODO")
}

type typeResolver struct {
	scalar string
	typ    schema.Type
}

func (r *typeResolver) Kind() string {
	if r.scalar != "" {
		return "SCALAR"
	}
	switch r.typ.(type) {
	case *schema.Object:
		return "OBJECT"
	case *schema.Interface:
		return "INTERFACE"
	case *schema.Union:
		return "UNION"
	case *schema.Enum:
		return "ENUM"
	case *schema.Input:
		return "INPUT_OBJECT"
	case *schema.List:
		return "LIST"
	// TODO NON_NULL
	default:
		panic("unreachable")
	}
}

func (r *typeResolver) Name() string {
	if r.scalar != "" {
		return r.scalar
	}
	switch t := r.typ.(type) {
	case *schema.Object:
		return t.Name
	case *schema.Interface:
		return t.Name
	case *schema.Union:
		return t.Name
	case *schema.Enum:
		return t.Name
	case *schema.Input:
		return t.Name
	default:
		panic("unreachable")
	}
}

func (r *typeResolver) Description() string {
	panic("TODO")
}

func (r *typeResolver) Fields() []*fieldResolver {
	panic("TODO")
}

func (r *typeResolver) Interfaces() []*typeResolver {
	panic("TODO")
}

func (r *typeResolver) PossibleTypes() []*typeResolver {
	panic("TODO")
}

func (r *typeResolver) EnumValues() []*enumValueResolver {
	panic("TODO")
}

func (r *typeResolver) InputFields() []*inputValueResolver {
	panic("TODO")
}

func (r *typeResolver) OfType() *typeResolver {
	panic("TODO")
}

type fieldResolver struct {
}

func (r *fieldResolver) Name() string {
	panic("TODO")
}

func (r *fieldResolver) Description() string {
	panic("TODO")
}

func (r *fieldResolver) Args() []*inputValueResolver {
	panic("TODO")
}

func (r *fieldResolver) Type() *typeResolver {
	panic("TODO")
}

func (r *fieldResolver) IsDeprecated() bool {
	panic("TODO")
}

func (r *fieldResolver) DeprecationReason() string {
	panic("TODO")
}

type inputValueResolver struct {
}

func (r *inputValueResolver) Name() string {
	panic("TODO")
}

func (r *inputValueResolver) Description() string {
	panic("TODO")
}

func (r *inputValueResolver) Type() *typeResolver {
	panic("TODO")
}

func (r *inputValueResolver) DefaultValue() string {
	panic("TODO")
}

type enumValueResolver struct {
}

func (r *enumValueResolver) Name() string {
	panic("TODO")
}

func (r *enumValueResolver) Description() string {
	panic("TODO")
}

func (r *enumValueResolver) IsDeprecated() bool {
	panic("TODO")
}

func (r *enumValueResolver) DeprecationReason() string {
	panic("TODO")
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

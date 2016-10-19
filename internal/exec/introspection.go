package exec

import (
	"context"
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
	return schemaExec.exec(r, selSet, reflect.ValueOf(&schemaResolver{r.schema}))
}

func introspectType(r *request, name string, selSet *query.SelectionSet) interface{} {
	t, ok := r.schema.AllTypes[name]
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

func (r *schemaResolver) Types(ctx context.Context) []*typeResolver {
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

func (r *schemaResolver) QueryType(ctx context.Context) *typeResolver {
	return &typeResolver{typ: r.schema.AllTypes[r.schema.EntryPoints["query"]]}
}

func (r *schemaResolver) MutationType(ctx context.Context) *typeResolver {
	return &typeResolver{typ: r.schema.AllTypes[r.schema.EntryPoints["mutation"]]}
}

func (r *schemaResolver) Directives(ctx context.Context) []*directiveResolver {
	panic("TODO")
}

type typeResolver struct {
	scalar string
	typ    schema.Type
}

func (r *typeResolver) Kind(ctx context.Context) string {
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

func (r *typeResolver) Name(ctx context.Context) string {
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
	case *schema.InputObject:
		return t.Name
	default:
		panic("unreachable")
	}
}

func (r *typeResolver) Description(ctx context.Context) string {
	panic("TODO")
}

func (r *typeResolver) Fields(ctx context.Context, args struct{ IncludeDeprecated bool }) []*fieldResolver {
	panic("TODO")
}

func (r *typeResolver) Interfaces(ctx context.Context) []*typeResolver {
	panic("TODO")
}

func (r *typeResolver) PossibleTypes(ctx context.Context) []*typeResolver {
	panic("TODO")
}

func (r *typeResolver) EnumValues(ctx context.Context, args struct{ IncludeDeprecated bool }) []*enumValueResolver {
	panic("TODO")
}

func (r *typeResolver) InputFields(ctx context.Context) []*inputValueResolver {
	panic("TODO")
}

func (r *typeResolver) OfType(ctx context.Context) *typeResolver {
	panic("TODO")
}

type fieldResolver struct {
}

func (r *fieldResolver) Name(ctx context.Context) string {
	panic("TODO")
}

func (r *fieldResolver) Description(ctx context.Context) string {
	panic("TODO")
}

func (r *fieldResolver) Args(ctx context.Context) []*inputValueResolver {
	panic("TODO")
}

func (r *fieldResolver) Type(ctx context.Context) *typeResolver {
	panic("TODO")
}

func (r *fieldResolver) IsDeprecated(ctx context.Context) bool {
	panic("TODO")
}

func (r *fieldResolver) DeprecationReason(ctx context.Context) string {
	panic("TODO")
}

type inputValueResolver struct {
}

func (r *inputValueResolver) Name(ctx context.Context) string {
	panic("TODO")
}

func (r *inputValueResolver) Description(ctx context.Context) string {
	panic("TODO")
}

func (r *inputValueResolver) Type(ctx context.Context) *typeResolver {
	panic("TODO")
}

func (r *inputValueResolver) DefaultValue(ctx context.Context) string {
	panic("TODO")
}

type enumValueResolver struct {
}

func (r *enumValueResolver) Name(ctx context.Context) string {
	panic("TODO")
}

func (r *enumValueResolver) Description(ctx context.Context) string {
	panic("TODO")
}

func (r *enumValueResolver) IsDeprecated(ctx context.Context) bool {
	panic("TODO")
}

func (r *enumValueResolver) DeprecationReason(ctx context.Context) string {
	panic("TODO")
}

type directiveResolver struct {
}

func (r *directiveResolver) Name(ctx context.Context) string {
	panic("TODO")
}

func (r *directiveResolver) Description(ctx context.Context) string {
	panic("TODO")
}

func (r *directiveResolver) Locations(ctx context.Context) []string {
	panic("TODO")
}

func (r *directiveResolver) Args(ctx context.Context) []*inputValueResolver {
	panic("TODO")
}

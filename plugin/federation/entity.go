package federation

import (
	"go/types"
	"slices"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin/federation/fieldset"
)

// RequiresStrategy selects how an entity's @requires fields are delivered to its
// resolver. Three strategies target the entity resolver and form a single axis —
// WHERE @requires data is handed over, relative to the resolver call — and are
// selected per entity via @entityResolver(requires: "…"), falling back to the
// package-level default:
//
//   - RequiresDefault ("default"): unmarshaled onto the returned entity AFTER the
//     resolver runs (output, after).
//   - RequiresExplicit ("explicit"): handed to a user-implemented
//     Populate<Entity>Requires hook as the raw representation map, AFTER the
//     resolver runs (output, after — user-owned). This is the only strategy that
//     surfaces the raw representation to user code.
//   - RequiresPreloaded ("preloaded"): unmarshaled onto the resolver's INPUT
//     representation BEFORE the resolver runs, so a multi resolver sees every
//     entity's @requires data in one scope (input, before). Multi entities only.
//
// RequiresComputed ("computed") is the outlier: it does not touch the entity
// resolver at all. It routes @requires to standalone field resolvers via a
// federationRequires argument (Federation 2 only), so it is off the axis the
// directive selects on — it is therefore NOT a @entityResolver(requires:) value.
// It is selected either per package (computed_requires, computing every @requires
// field on its entities) or per field (@computedRequires on FIELD_DEFINITION), which
// lets one entity compute some @requires fields while delivering the rest through
// the entity resolver. The per-field flag lives on Requires.Computed. It shares
// this type because the entity-resolver strategies are mutually exclusive per
// entity.
//
// The entity-resolver strategies (default/explicit/preloaded) are mutually
// exclusive: each entity resolves to exactly one.
type RequiresStrategy string

const (
	// RequiresDefault unmarshals @requires onto the returned entity after the
	// resolver runs.
	RequiresDefault RequiresStrategy = "default"
	// RequiresExplicit delegates @requires population to a user-implemented
	// Populate<Entity>Requires function, called after the resolver.
	RequiresExplicit RequiresStrategy = "explicit"
	// RequiresComputed delivers @requires to standalone field resolvers
	// (Federation 2 only). Selected by the computed_requires package option,
	// not by @entityResolver(requires:).
	RequiresComputed RequiresStrategy = "computed"
	// RequiresPreloaded unmarshals @requires onto the resolver's input
	// representation before the resolver runs, so a multi resolver sees every
	// entity's @requires data at once. Multi entities only.
	RequiresPreloaded RequiresStrategy = "preloaded"
)

// Entity represents a federated type
// that was declared in the GQL schema.
type Entity struct {
	Name      string // The same name as the type declaration
	Def       *ast.Definition
	Resolvers []*EntityResolver
	Requires  []*Requires
	Multi     bool
	// RequiresStrategy is how this entity's @requires fields are delivered to
	// the resolver. Resolved per entity in buildEntity.
	RequiresStrategy RequiresStrategy
	Type             types.Type
	// ImplDirectives are the resolved non-federation OBJECT-level directives
	// with full type information, populated in GenerateCode for use in the
	// federation template to wrap entity resolver calls.
	ImplDirectives []*codegen.Directive
}

// IsDefaultRequires reports whether @requires uses the default (post-resolver
// unmarshal) strategy.
func (e *Entity) IsDefaultRequires() bool { return e.RequiresStrategy == RequiresDefault }

// IsExplicitRequires reports whether @requires uses a user Populate function.
func (e *Entity) IsExplicitRequires() bool { return e.RequiresStrategy == RequiresExplicit }

// IsPreloaded reports whether @requires is populated onto the resolver input
// representation before the resolver runs.
func (e *Entity) IsPreloaded() bool {
	return e.RequiresStrategy == RequiresPreloaded
}

type EntityResolver struct {
	ResolverName   string      // The resolver name, such as FindUserByID
	KeyFields      []*KeyField // The fields declared in @key.
	InputType      types.Type  // The Go generated input type for multi entity resolvers
	InputTypeName  string
	ReturnType     types.Type // The Go generated return type for the entity
	ReturnTypeName string
}

func (e *EntityResolver) LookupInputType() string {
	return templates.CurrentImports.LookupType(e.InputType)
}

func (e *EntityResolver) LookupReturnType() string {
	return templates.CurrentImports.LookupType(e.ReturnType)
}

// IsPointerReturnType returns true if the resolver's return type is a pointer
func (e *EntityResolver) IsPointerReturnType() bool {
	if e.ReturnType == nil {
		return false
	}
	lookupType := templates.CurrentImports.LookupType(e.ReturnType)
	return lookupType != "" && lookupType[0] == '*'
}

type KeyField struct {
	Definition *ast.FieldDefinition
	Field      fieldset.Field        // len > 1 for nested fields
	Type       *config.TypeReference // The Go representation of that field type
	// GoName is the field name this key takes in the generated multi-resolver
	// input struct. It is normally Field.ToGo(), but is disambiguated with a
	// numeric suffix when two key paths in the same resolver would otherwise
	// produce the same Go name (e.g. "id" and "i { d }" both yield "ID").
	// Using a single stored name keeps the SDL input field, the modelgen struct
	// field, and the template's struct literal in agreement.
	GoName string
}

// Requires represents an @requires clause
type Requires struct {
	Name  string                // the name of the field
	Field fieldset.Field        // source Field, len > 1 for nested fields
	Type  *config.TypeReference // The Go representation of that field type
	// Computed reports whether this @requires field is delivered via a standalone
	// field resolver (the computed strategy) rather than through the entity
	// resolver. It is true when the field carries @computedRequires, or when the whole
	// entity resolves to RequiresComputed (the computed_requires package option,
	// which computes every @requires field). Set in buildRequires.
	Computed bool
}

func (e *Entity) allFieldsAreExternal(federationVersion int) bool {
	for _, field := range e.Def.Fields {
		if !e.isFieldImplicitlyExternal(field, federationVersion) &&
			field.Directives.ForName("external") == nil {
			return false
		}
	}
	return true
}

// In federation v2, key fields are implicitly external.
func (e *Entity) isFieldImplicitlyExternal(field *ast.FieldDefinition, federationVersion int) bool {
	// Key fields are only implicitly external in Federation 2
	if federationVersion != 2 {
		return false
	}
	// TODO: From the spec, it seems like if an entity is not resolvable then it should not only not
	// have a resolver, but should not appear in the _Entity union. The current implementation is a
	// less drastic departure from the previous behavior, but should probably be reviewed.
	// See https://www.apollographql.com/docs/federation/subgraph-spec/
	if e.isResolvable() {
		return false
	}
	// If the field is a key field, it is implicitly external
	if e.isKeyField(field) {
		return true
	}

	return false
}

// Determine if the entity is resolvable.
func (e *Entity) isResolvable() bool {
	key := e.Def.Directives.ForName(dirNameKey)
	if key == nil {
		// If there is no key directive, the entity is resolvable.
		return true
	}
	resolvable := key.Arguments.ForName("resolvable")
	if resolvable == nil {
		// If there is no resolvable argument, the entity is resolvable.
		return true
	}
	// only if resolvable: false has been set on the @key directive do we consider the entity
	// non-resolvable.
	return resolvable.Value.Raw != "false"
}

// Determine if a field is part of the entities key.
func (e *Entity) isKeyField(field *ast.FieldDefinition) bool {
	return slices.Contains(e.keyFields(), field.Name)
}

// Get the key fields for this entity.
func (e *Entity) keyFields() []string {
	key := e.Def.Directives.ForName(dirNameKey)
	if key == nil {
		return []string{}
	}
	fields := key.Arguments.ForName(DirArgFields)
	if fields == nil {
		return []string{}
	}
	fieldSet := fieldset.New(fields.Value.Raw, nil)
	keyFields := make([]string, len(fieldSet))
	for i, field := range fieldSet {
		keyFields[i] = field[0]
	}
	return keyFields
}

// GetTypeInfo - get the imported package & type name combo.  package.TypeName
func (e Entity) GetTypeInfo() string {
	return templates.CurrentImports.LookupType(e.Type)
}

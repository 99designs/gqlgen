package federation

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
)

// A representation is a generated input object that carries an entity's @key
// fields, and with the preloaded strategy its @requires fields, to its
// per-entity resolver. Every field is nullable, so a field the router did not
// send arrives as nil. A nested selection such as "owner { id }" gets its own
// representation of the nested type, shared by every per-entity entity that
// selects into that type, so it lists the fields all of them select.
type representation struct {
	// Name is the GraphQL input name: the owner's representation_names entry,
	// else owner type + "Representation".
	Name   string
	Owner  string // the type whose fields this mirrors
	Fields []*representationField
	// Root marks an entity's own representation, the resolver's argument,
	// which reports the key that matched.
	Root bool
}

type representationField struct {
	Name string // the owner's GraphQL field name
	// GoName is the representation's own fieldName in gqlgen.yml, else the
	// Go name of the owner's field (goFieldName).
	GoName string
	Type   string // SDL type: the owner's field type made nullable
}

// representationNamesSetting is the gqlgen.yml setting that names
// representations, quoted in errors.
const representationNamesSetting = "federation.representation_names"

// rootRepresentation returns the representation that is entity's resolver
// argument.
//
// Requires: representations come from buildRepresentations, with entity
// per-entity.
func rootRepresentation(representations []*representation, entity *Entity) *representation {
	for _, r := range representations {
		if r.Root && r.Owner == entity.Name {
			return r
		}
	}
	return nil
}

// SDL renders the representation as an input object definition.
func (r *representation) SDL() string {
	var sb strings.Builder
	sb.WriteString("input " + r.Name + " {\n")
	for _, field := range r.Fields {
		sb.WriteString("\t" + field.Name + ": " + field.Type + "\n")
	}
	sb.WriteString("}")
	return sb.String()
}

// perEntityResolverSDL renders the entity's single resolver field: one
// representation for a single resolver, a list of them for a batch resolver.
func perEntityResolverSDL(e *Entity, rep string) string {
	if e.Multi {
		return fmt.Sprintf("\t%s(reps: [%s!]!): [%s]", e.PerEntityResolverName(), rep, e.Name)
	}
	return fmt.Sprintf("\t%s(rep: %s!): %s!", e.PerEntityResolverName(), rep, e.Name)
}

// buildRepresentations derives the representation inputs for every per-entity
// entity from its @key fields and, for a preloaded entity, its non-computed
// @requires fields.
//
// Requires: entities are sorted and their key fields exist in schema; names
// is federation.representation_names.
// Ensures:  one representation per per-entity entity (Root) and per type its
// selections reach, in first-use order, named from names or by default, each
// listing the selected fields once, keys first, with their Go names; or an
// error if a selection is one add rejects, a representation's Go name is a
// schema type's or another representation's, or names has an entry for no
// representation. Two fields sharing a Go name are left to modelgen, which
// checks every generated struct after its hooks have run.
func buildRepresentations(
	entities []*Entity,
	schema *ast.Schema,
	models config.TypeMap,
	names map[string]string,
) ([]*representation, error) {
	b := representationBuilder{
		schema:  schema,
		models:  models,
		names:   names,
		goTypes: goTypeNames(schema),
		byGo:    make(map[string]*representation),
		byOwner: make(map[string]*representation),
	}
	for _, entity := range entities {
		if !entity.PerEntity {
			continue
		}
		if err := b.addEntity(entity); err != nil {
			return nil, fmt.Errorf("entity %q: %w", entity.Name, err)
		}
	}
	for _, owner := range slices.Sorted(maps.Keys(names)) {
		if b.byOwner[owner] == nil {
			return nil, fmt.Errorf(
				"%s: no representation is generated for %q", representationNamesSetting, owner)
		}
	}
	return b.ordered, nil
}

type representationBuilder struct {
	schema  *ast.Schema
	models  config.TypeMap
	names   map[string]string // representation_names: owner to name
	goTypes map[string]string // Go name to schema type name
	byGo    map[string]*representation
	byOwner map[string]*representation
	ordered []*representation
}

// addEntity records the representations of entity's @key fields and, when
// preloaded, of the @requires fields its resolver receives.
func (b *representationBuilder) addEntity(entity *Entity) error {
	root, err := b.representationOf(entity.Def)
	if err != nil {
		return err
	}
	root.Root = true
	for _, resolver := range entity.Resolvers {
		for _, keyField := range resolver.KeyFields {
			if err := b.add(root, entity.Def, keyField.Field, keyPath); err != nil {
				return err
			}
		}
	}
	if !entity.IsPreloaded() {
		return nil
	}
	for _, req := range entity.Requires {
		if req.Computed {
			continue // resolved by its field resolver, not the entity resolver
		}
		if err := b.add(root, entity.Def, req.Field, requiresPath); err != nil {
			return err
		}
	}
	return nil
}

// representationOf returns def's representation, creating it on first use.
func (b *representationBuilder) representationOf(def *ast.Definition) (*representation, error) {
	if r, ok := b.byOwner[def.Name]; ok {
		return r, nil
	}
	name, pinned := b.names[def.Name]
	if !pinned {
		name = def.Name + "Representation"
	}
	goName := templates.ToGo(name)
	if existing, ok := b.goTypes[goName]; ok {
		if pinned {
			return nil, fmt.Errorf(
				"type %q has the same Go name as representation %s, named in %s; "+
					"choose another name", existing, name, representationNamesSetting)
		}
		return nil, fmt.Errorf(
			"type %q has the same Go name as representation %s; rename %q, "+
				"or name the representation of %q in %s",
			existing, name, existing, def.Name, representationNamesSetting)
	}
	if other, ok := b.byGo[goName]; ok {
		return nil, fmt.Errorf(
			"the representations of %q and %q both have Go name %s; "+
				"give one another name in %s",
			other.Owner, def.Name, goName, representationNamesSetting)
	}
	r := &representation{Name: name, Owner: def.Name}
	b.byGo[goName] = r
	b.byOwner[def.Name] = r
	b.ordered = append(b.ordered, r)
	return r, nil
}

// pathKind is the directive a representation path comes from, as errors
// name it.
type pathKind string

const (
	keyPath      pathKind = "key"
	requiresPath pathKind = "@requires"
)

// add records path on r, whose owner is def, creating nested representations
// for each object or interface the path passes through; a list stays a list
// of the nested representation.
//
// Ensures: the schema's types are not modified; or an error if the path
// names no field, selects an object without its fields, passes through an
// inline fragment, or, as a key, selects a list, interface or union field,
// which federation does not allow in @key.
func (b *representationBuilder) add(
	r *representation,
	def *ast.Definition,
	path []string,
	kind pathKind,
) error {
	fieldDef := def.Fields.ForName(path[0])
	if fieldDef == nil {
		return fmt.Errorf("%s field %q is not a field of %s", kind, path[0], def.Name)
	}
	typeDef := b.schema.Types[fieldDef.Type.Name()]

	if kind == keyPath && (fieldDef.Type.Elem != nil ||
		typeDef.Kind == ast.Interface || typeDef.Kind == ast.Union) {
		return fmt.Errorf(
			"key field %s.%s has type %s; federation does not allow list, "+
				"interface or union fields in @key",
			def.Name, fieldDef.Name, fieldDef.Type,
		)
	}
	if len(path) == 1 && typeDef.Kind != ast.Scalar && typeDef.Kind != ast.Enum {
		return fmt.Errorf(
			"%s field %s.%s has %s type %s; select its fields",
			kind, def.Name, fieldDef.Name, strings.ToLower(string(typeDef.Kind)), typeDef.Name,
		)
	}

	typeName := typeDef.Name
	if len(path) > 1 {
		if strings.HasPrefix(path[1], "...") {
			return fmt.Errorf(
				"%s path through inline fragment %q on %s.%s is not supported with the %s option",
				kind, path[1], def.Name, fieldDef.Name, optionEntityResolverPerEntity,
			)
		}
		nested, err := b.representationOf(typeDef)
		if err != nil {
			return err
		}
		if err := b.add(nested, typeDef, path[1:], kind); err != nil {
			return err
		}
		typeName = nested.Name
	}
	fieldType := withNamedType(fieldDef.Type, typeName)
	fieldType.NonNull = false

	if slices.ContainsFunc(r.Fields, func(field *representationField) bool {
		return field.Name == fieldDef.Name
	}) {
		return nil // another path selected it, perhaps with other nested fields
	}
	goName := b.models[r.Name].Fields[fieldDef.Name].FieldName
	if goName == "" {
		goName = goFieldName(b.models, def.Name, fieldDef)
	}
	r.Fields = append(r.Fields, &representationField{
		Name:   fieldDef.Name,
		GoName: goName,
		Type:   fieldType.String(),
	})
	return nil
}

// matchedKeyField is the Go field of a root representation that reports the
// @key the representation matched; federation.gotpl sets it.
const matchedKeyField = "MatchedKey"

// configureRepresentationModels records in models what modelgen needs to
// generate each representation: its fields' Go names, as
// buildRepresentations chose them, and, on a root representation, the
// matchedKeyField extra field alongside any extraFields of the user's.
//
// Ensures: nil, or an error if the user's extraFields already define
// matchedKeyField on a root representation.
func configureRepresentationModels(representations []*representation, models config.TypeMap) error {
	for _, r := range representations {
		entry := models[r.Name]
		if entry.Fields == nil {
			entry.Fields = make(map[string]config.TypeMapField, len(r.Fields))
		}
		for _, field := range r.Fields {
			fieldEntry := entry.Fields[field.Name]
			fieldEntry.FieldName = field.GoName
			entry.Fields[field.Name] = fieldEntry
		}
		if r.Root {
			if _, taken := entry.ExtraFields[matchedKeyField]; taken {
				return fmt.Errorf(
					"models.%s.extraFields.%s is reserved: gqlgen sets it to the @key "+
						"the representation matched", r.Name, matchedKeyField)
			}
			if entry.ExtraFields == nil {
				entry.ExtraFields = make(map[string]config.ModelExtraField, 1)
			}
			entry.ExtraFields[matchedKeyField] = config.ModelExtraField{
				Type:        "string",
				Description: "The @key fields this representation matched, as gqlgen prints them.",
			}
		}
		models[r.Name] = entry
	}
	return nil
}

// withNamedType returns a copy of t, down to its innermost named type, naming
// that type name; t itself, which belongs to the schema, is unchanged.
func withNamedType(t *ast.Type, name string) *ast.Type {
	c := *t
	if c.Elem != nil {
		c.Elem = withNamedType(c.Elem, name)
	} else {
		c.NamedType = name
	}
	return &c
}

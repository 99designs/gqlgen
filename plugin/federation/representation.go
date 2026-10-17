package federation

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
)

// A representation is a generated input object that carries an entity's @key
// fields to its per-entity resolver. Every field is nullable, so a key the
// router did not send arrives as nil. A nested key selection such as
// "owner { id }" gets its own representation of the nested type, shared by
// every per-entity entity that selects into that type.
type representation struct {
	Name   string // GraphQL input name: owner type + "Representation"
	Owner  string // the type whose fields this mirrors
	Fields []*representationField
}

type representationField struct {
	Name string // the owner's GraphQL field name
	Type string // SDL type: the owner's field type made nullable
}

func representationName(typeName string) string {
	return typeName + "Representation"
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
func perEntityResolverSDL(e *Entity) string {
	rep := representationName(e.Name)
	if e.Multi {
		return fmt.Sprintf("\t%s(reps: [%s!]!): [%s]", e.PerEntityResolverName(), rep, e.Name)
	}
	return fmt.Sprintf("\t%s(rep: %s!): %s!", e.PerEntityResolverName(), rep, e.Name)
}

// buildRepresentations derives the representation inputs for every per-entity
// entity from its @key fields.
//
// Requires: entities are sorted and their key fields exist in schema.
// Ensures:  one representation per per-entity entity and per type its keys
// select into, in first-use order, each listing the selected fields once in
// key order; an error if a representation name is already a schema type.
func buildRepresentations(entities []*Entity, schema *ast.Schema) ([]*representation, error) {
	b := representationBuilder{schema: schema, byName: make(map[string]*representation)}
	for _, entity := range entities {
		if !entity.PerEntity {
			continue
		}
		root, err := b.representationOf(entity.Def)
		if err != nil {
			return nil, fmt.Errorf("entity %q: %w", entity.Name, err)
		}
		for _, resolver := range entity.Resolvers {
			for _, keyField := range resolver.KeyFields {
				if err := b.add(root, entity.Def, keyField.Field); err != nil {
					return nil, fmt.Errorf("entity %q: %w", entity.Name, err)
				}
			}
		}
	}
	return b.ordered, nil
}

type representationBuilder struct {
	schema  *ast.Schema
	byName  map[string]*representation
	ordered []*representation
}

func (b *representationBuilder) representationOf(def *ast.Definition) (*representation, error) {
	name := representationName(def.Name)
	if r, ok := b.byName[name]; ok {
		return r, nil
	}
	if b.schema.Types[name] != nil {
		return nil, fmt.Errorf("type %s already exists", name)
	}
	r := &representation{Name: name, Owner: def.Name}
	b.byName[name] = r
	b.ordered = append(b.ordered, r)
	return r, nil
}

// add records the key path on r, whose owner is def, creating nested
// representations for each object the path passes through.
func (b *representationBuilder) add(r *representation, def *ast.Definition, path []string) error {
	fieldDef := def.Fields.ForName(path[0])
	fieldType := *fieldDef.Type
	fieldType.NonNull = false

	if len(path) > 1 {
		if fieldType.Elem != nil {
			return fmt.Errorf(
				"key path through list field %s.%s is not supported",
				def.Name,
				fieldDef.Name,
			)
		}
		nestedDef := b.schema.Types[fieldType.NamedType]
		nested, err := b.representationOf(nestedDef)
		if err != nil {
			return err
		}
		if err := b.add(nested, nestedDef, path[1:]); err != nil {
			return err
		}
		fieldType.NamedType = nested.Name
	}

	for _, field := range r.Fields {
		if field.Name == fieldDef.Name {
			return nil
		}
	}
	r.Fields = append(r.Fields, &representationField{Name: fieldDef.Name, Type: fieldType.String()})
	return nil
}

// copyRepresentationFieldNames gives each representation field the Go name its
// owner's field has, so a key such as _id with @goField(name: "LegacyID")
// lands on LegacyID in both structs, and colliding keys stay distinct.
//
// Requires: models already holds @goField and fieldName overrides (config.Init).
// Ensures:  a representation field gets its owner's FieldName unless the user
// set one on the representation itself; nothing else changes.
func copyRepresentationFieldNames(representations []*representation, models config.TypeMap) {
	for _, r := range representations {
		for _, field := range r.Fields {
			name := models[r.Owner].Fields[field.Name].FieldName
			if name == "" {
				continue
			}
			entry := models[r.Name]
			if entry.Fields[field.Name].FieldName != "" {
				continue
			}
			if entry.Fields == nil {
				entry.Fields = make(map[string]config.TypeMapField)
			}
			fieldEntry := entry.Fields[field.Name]
			fieldEntry.FieldName = name
			entry.Fields[field.Name] = fieldEntry
			models[r.Name] = entry
		}
	}
}

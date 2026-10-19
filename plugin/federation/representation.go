package federation

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
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
	// GoName is the representation's own fieldName in gqlgen.yml, else the
	// Go name of the owner's field (goFieldName).
	GoName string
	Type   string // SDL type: the owner's field type made nullable
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
// key order with their Go names; or an error if a representation's Go name is
// a schema type's, or two of its fields share a Go name.
func buildRepresentations(
	entities []*Entity,
	schema *ast.Schema,
	models config.TypeMap,
) ([]*representation, error) {
	b := representationBuilder{
		schema:  schema,
		models:  models,
		goTypes: goTypeNames(schema),
		byName:  make(map[string]*representation),
	}
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
	models  config.TypeMap
	goTypes map[string]string // Go name to schema type name
	byName  map[string]*representation
	ordered []*representation
}

func (b *representationBuilder) representationOf(def *ast.Definition) (*representation, error) {
	name := representationName(def.Name)
	if r, ok := b.byName[name]; ok {
		return r, nil
	}
	if existing, ok := b.goTypes[templates.ToGo(name)]; ok {
		return nil, fmt.Errorf(
			"type %q has the same Go name as representation %s; rename %q",
			existing, name, existing,
		)
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

	goName := b.models[r.Name].Fields[fieldDef.Name].FieldName
	if goName == "" {
		goName = goFieldName(b.models, def.Name, fieldDef)
	}
	for _, field := range r.Fields {
		if field.Name == fieldDef.Name {
			return nil
		}
		// Exact, not by ToGo: modelgen uses a Go name as written.
		if field.GoName == goName {
			return fmt.Errorf(
				"%s.%s and %s.%s both have Go name %s in %s; give one a distinct "+
					"@goField(name: ...) or gqlgen.yml fieldName",
				def.Name, field.Name, def.Name, fieldDef.Name, goName, r.Name,
			)
		}
	}
	r.Fields = append(r.Fields, &representationField{
		Name:   fieldDef.Name,
		GoName: goName,
		Type:   fieldType.String(),
	})
	return nil
}

// setRepresentationFieldNames records each representation field's Go name in
// models, so the generated struct has the names buildRepresentations checked.
func setRepresentationFieldNames(representations []*representation, models config.TypeMap) {
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
		models[r.Name] = entry
	}
}

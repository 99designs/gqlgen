package federation

import (
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/plugin/federation/fieldset"
	"github.com/vektah/gqlparser/v2/ast"
)

// Entity represents a federated type
// that was declared in the GQL schema.
type Entity struct {
	Name      string // The same name as the type declaration
	Def       *ast.Definition
	Resolvers []*EntityResolver
	Requires  []*Requires
	Multi     bool
}

type EntityResolver struct {
	ResolverName string      // The resolver name, such as FindUserByID
	KeyFields    []*KeyField // The fields declared in @key.
	InputType    string      // The Go generated input type for multi entity resolvers
}

type KeyField struct {
	Definition *ast.FieldDefinition
	Field      fieldset.Field        // len > 1 for nested fields
	Type       *config.TypeReference // The Go representation of that field type
}

// Requires represents an @requires clause
type Requires struct {
	Name  string                // the name of the field
	Field fieldset.Field        // source Field, len > 1 for nested fields
	Type  *config.TypeReference // The Go representation of that field type
}

func (e *Entity) allFieldsAreExternal(federationVersion int) bool {
	for _, field := range e.Def.Fields {
		if field.Directives.ForName("external") == nil {
			if e.isFieldImplicitlyExternal(field, federationVersion) {
				continue
			}
			return false
		}
	}
	return true
}

// In federation v2, key fields are implicitly external.
func (e *Entity) isFieldImplicitlyExternal(field *ast.FieldDefinition, federationVersion int) bool {
	// Key fields are only implicitly external in Federation 2
	if federationVersion == 1 {
		return false
	}
	// Only allow non resolvable entities to have implicit external fields.
	// This may not be required, but it's a safer incase there are edge cases.
	if e.isResolvable() {
		return false
	}
	// If the field is a key field, it is implicitly external
	if e.isKeyField(field) {
		return true
	}

	return false
}

func (e *Entity) isResolvable() bool {
	key := e.Def.Directives.ForName("key")
	if key == nil {
		return true
	}
	resolvable := key.Arguments.ForName("resolvable")
	if resolvable == nil {
		return true
	}
	return resolvable.Value.Raw != "false"
}

// Determine if a field is part of the entity's key.
func (e *Entity) isKeyField(field *ast.FieldDefinition) bool {
	for _, keyField := range e.keyFields() {
		if keyField == field.Name {
			return true
		}
	}
	return false
}

// Get the key fields for this entity.
func (e *Entity) keyFields() []string {
	var keyFields []string
	key := e.Def.Directives.ForName("key")
	if key == nil {
		return keyFields
	}
	fields := key.Arguments.ForName("fields")
	if fields == nil {
		return keyFields
	}
	for _, field := range fieldset.New(fields.Value.Raw, nil) {
		keyFields = append(keyFields, field[0])
	}
	return keyFields
}

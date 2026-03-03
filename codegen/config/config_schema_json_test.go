package config

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jsonSchemaProperty represents a node in a JSON Schema tree, capturing
// the three ways sub-fields can be described: direct properties, map-like
// additionalProperties, and array items.
type jsonSchemaProperty struct {
	Properties           map[string]jsonSchemaProperty `json:"properties"`
	AdditionalProperties *jsonSchemaProperty           `json:"additionalProperties"`
	Items                *jsonSchemaProperty           `json:"items"`
}

// extractYAMLTagName returns the yaml tag name for a struct field,
// or "" if the field should be skipped (no tag, "-", or empty name).
func extractYAMLTagName(field reflect.StructField) string {
	tag := field.Tag.Get("yaml")
	if tag == "" || tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

// resolveSchemaProps determines which JSON Schema property map should be used
// to validate sub-fields for the given Go type:
//   - struct / *struct       → properties
//   - map[K]struct           → additionalProperties.properties
//   - []struct               → items.properties
func resolveSchemaProps(
	goType reflect.Type,
	prop jsonSchemaProperty,
) (structType reflect.Type, schemaProps map[string]jsonSchemaProperty) {
	// Unwrap pointer(s).
	for goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}

	switch goType.Kind() {
	case reflect.Struct:
		return goType, prop.Properties

	case reflect.Map:
		valType := goType.Elem()
		for valType.Kind() == reflect.Ptr {
			valType = valType.Elem()
		}
		if valType.Kind() == reflect.Struct && prop.AdditionalProperties != nil {
			return valType, prop.AdditionalProperties.Properties
		}

	case reflect.Slice:
		elemType := goType.Elem()
		for elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		if elemType.Kind() == reflect.Struct && prop.Items != nil {
			return elemType, prop.Items.Properties
		}
	}

	return nil, nil
}

// checkStructFieldsInSchema recursively verifies that every yaml-tagged field
// in structType has a corresponding key in schemaProps, then recurses into
// nested structs, maps-with-struct-values, and slices-of-structs.
func checkStructFieldsInSchema(
	t *testing.T,
	structType reflect.Type,
	schemaProps map[string]jsonSchemaProperty,
	path string,
) {
	t.Helper()
	if len(schemaProps) == 0 {
		return
	}

	for i := range structType.NumField() {
		field := structType.Field(i)
		yamlName := extractYAMLTagName(field)
		if yamlName == "" {
			continue
		}

		assert.Contains(t, schemaProps, yamlName,
			"%s.%s (yaml:%q) is missing from gqlgen.schema.json at path %s",
			structType.Name(), field.Name, yamlName, path)

		prop, ok := schemaProps[yamlName]
		if !ok {
			continue
		}

		// Recurse into nested types.
		childStruct, childProps := resolveSchemaProps(field.Type, prop)
		if childStruct != nil && len(childProps) > 0 {
			checkStructFieldsInSchema(t, childStruct, childProps, path+"."+yamlName)
		}
	}
}

// TestConfigFieldsPresentInSchemaJSON verifies that every yaml-tagged field
// in the Config struct (and all nested structs, map-value structs, slice-element
// structs) has a corresponding property in the gqlgen.schema.json file.
//
// All nested sections are discovered via reflection — no manual list needed.
func TestConfigFieldsPresentInSchemaJSON(t *testing.T) {
	schemaPath := "../../gqlgen.schema.json"
	data, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "failed to read gqlgen.schema.json")

	var schema jsonSchemaProperty
	require.NoError(t, json.Unmarshal(data, &schema), "failed to parse gqlgen.schema.json")

	// Deprecated fields we intentionally do NOT require in the schema.
	deprecated := map[string]bool{
		"federated": true,
	}

	configType := reflect.TypeOf(Config{})
	for i := range configType.NumField() {
		field := configType.Field(i)
		yamlName := extractYAMLTagName(field)
		if yamlName == "" || deprecated[yamlName] {
			continue
		}

		assert.Contains(t, schema.Properties, yamlName,
			"Config.%s (yaml:%q) is missing from gqlgen.schema.json top-level properties",
			field.Name, yamlName)

		prop, ok := schema.Properties[yamlName]
		if !ok {
			continue
		}

		// Recurse into nested struct / map / slice types.
		childStruct, childProps := resolveSchemaProps(field.Type, prop)
		if childStruct != nil && len(childProps) > 0 {
			checkStructFieldsInSchema(t, childStruct, childProps, yamlName)
		}
	}
}

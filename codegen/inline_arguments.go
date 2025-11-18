package codegen

import (
	"bytes"
	"fmt"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
)

// InlineArgsInfo stores metadata about arguments that were inlined.
// Used during codegen to bundle expanded arguments back into a single resolver parameter.
type InlineArgsInfo struct {
	OriginalArgName string
	OriginalType    string
	OriginalASTType *ast.Type
	GoType          string
	ExpandedArgs    []string
}

// inlineArgsMetadata maps "TypeName.FieldName" to inline args metadata.
var inlineArgsMetadata = make(map[string]*InlineArgsInfo)

// ExpandInlineArguments expands arguments marked with @inlineArguments
// and stores metadata for later codegen phase.
func ExpandInlineArguments(schema *ast.Schema) error {
	for typeName, typeDef := range schema.Types {
		if typeDef.Kind != ast.Object && typeDef.Kind != ast.Interface {
			continue
		}

		for _, field := range typeDef.Fields {
			var inlinedIndices []int
			var expandedArguments [][]*ast.ArgumentDefinition

			for i, arg := range field.Arguments {
				if arg.Directives.ForName("inlineArguments") == nil {
					continue
				}

				argTypeName := arg.Type.Name()
				inputType := schema.Types[argTypeName]
				if inputType == nil {
					return fmt.Errorf(
						"@inlineArguments on %s.%s(%s): type %s not found in schema",
						typeName, field.Name, arg.Name, argTypeName,
					)
				}

				if inputType.Kind != ast.InputObject {
					return fmt.Errorf(
						"@inlineArguments on %s.%s(%s): type %s must be an INPUT_OBJECT (input types only), got %s. The directive can only expand input object types into individual arguments",
						typeName,
						field.Name,
						arg.Name,
						argTypeName,
						inputType.Kind,
					)
				}

				var expanded []*ast.ArgumentDefinition
				var expandedNames []string

				for _, inputField := range inputType.Fields {
					expandedArg := &ast.ArgumentDefinition{
						Name:         inputField.Name,
						Type:         inputField.Type,
						Description:  inputField.Description,
						DefaultValue: inputField.DefaultValue,
						Directives:   inputField.Directives,
						Position:     inputField.Position,
					}
					expanded = append(expanded, expandedArg)
					expandedNames = append(expandedNames, inputField.Name)
				}

				goType := argTypeName
				if goModelDir := inputType.Directives.ForName("goModel"); goModelDir != nil {
					if modelArg := goModelDir.Arguments.ForName("model"); modelArg != nil {
						if modelValue, err := modelArg.Value.Value(nil); err == nil {
							goType = modelValue.(string)
						}
					}
				}

				key := fmt.Sprintf("%s.%s", typeName, field.Name)
				inlineArgsMetadata[key] = &InlineArgsInfo{
					OriginalArgName: arg.Name,
					OriginalType:    argTypeName,
					OriginalASTType: arg.Type,
					GoType:          goType,
					ExpandedArgs:    expandedNames,
				}

				inlinedIndices = append(inlinedIndices, i)
				expandedArguments = append(expandedArguments, expanded)
			}

			if len(inlinedIndices) > 0 {
				var newArgs ast.ArgumentDefinitionList

				for i, arg := range field.Arguments {
					inlinedIdx := -1
					for idx, inlined := range inlinedIndices {
						if inlined == i {
							inlinedIdx = idx
							break
						}
					}

					if inlinedIdx >= 0 {
						newArgs = append(newArgs, expandedArguments[inlinedIdx]...)
					} else {
						newArgs = append(newArgs, arg)
					}
				}

				field.Arguments = newArgs
			}
		}
	}

	return nil
}

// GetInlineArgsMetadata retrieves metadata for a given type and field.
func GetInlineArgsMetadata(typeName, fieldName string) *InlineArgsInfo {
	key := fmt.Sprintf("%s.%s", typeName, fieldName)
	return inlineArgsMetadata[key]
}

// ClearInlineArgsMetadata clears all stored metadata.
func ClearInlineArgsMetadata() {
	inlineArgsMetadata = make(map[string]*InlineArgsInfo)
}

func SerializeTransformedSchema(
	schema *ast.Schema,
	originalSources []*ast.Source,
) ([]*ast.Source, error) {
	if len(inlineArgsMetadata) == 0 {
		return originalSources, nil
	}

	var buf bytes.Buffer
	f := formatter.NewFormatter(&buf)
	f.FormatSchema(schema)

	return []*ast.Source{
		{
			Name:    "inline_arguments_transformed_schema.graphql",
			Input:   buf.String(),
			BuiltIn: true,
		},
	}, nil
}

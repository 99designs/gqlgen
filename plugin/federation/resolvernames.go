package federation

import (
	"fmt"
	"slices"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin/federation/fieldset"
)

// resolverNames returns the resolver name for a key, such as findPortByID or
// findManyPortByIDs, and the batch input name, such as PortByIDsInput, with
// each key field named by keyName.
func resolverNames(
	entity string,
	keyFields []*KeyField,
	keyName func(fieldset.Field) string,
	multi bool,
) (resolver, input string) {
	names := make([]string, len(keyFields))
	for i, keyField := range keyFields {
		names[i] = keyName(keyField.Field)
	}
	stem := entity + "By" + strings.Join(names, "And")
	if multi {
		stem += "s" // Pluralize for better API readability
		return "findMany" + stem, stem + "Input"
	}
	return "find" + stem, stem + "Input"
}

// boundKeyName names a key path after the Go fields it binds to, segment by
// segment; see goFieldName.
//
// Requires: every segment of field is a field of parent or of the type the
// previous segment returns, as buildKeyFields checks.
func boundKeyName(
	models config.TypeMap,
	parent *ast.Definition,
	schema *ast.Schema,
	field fieldset.Field,
) string {
	var sb strings.Builder
	def := parent
	for _, segment := range field {
		fieldDef := def.Fields.ForName(segment)
		sb.WriteString(goFieldName(models, def.Name, fieldDef))
		def = schema.Types[fieldDef.Type.Name()]
	}
	return sb.String()
}

// goFieldName returns the Go name a field of typeName binds to: its
// @goField(name:), else its fieldName in gqlgen.yml, else its GraphQL name in
// Go form. That is the precedence config.Init applies, which runs only after
// resolvers are named.
func goFieldName(models config.TypeMap, typeName string, def *ast.FieldDefinition) string {
	if dir := def.Directives.ForName(config.DirGoField); dir != nil {
		if arg := dir.Arguments.ForName(config.DirArgName); arg != nil &&
			arg.Value.Kind == ast.StringValue {
			return arg.Value.Raw
		}
	}
	if name := models[typeName].Fields[def.Name].FieldName; name != "" {
		return name
	}
	return templates.ToGo(def.Name)
}

func sameKeyPath(a, b *KeyField) bool {
	return slices.Equal(a.Field, b.Field)
}

// checkResolverNames rejects generated names that gqlgen cannot change
// without renaming a type: entity names are fixed across subgraphs, and the
// clashing schema type belongs to the user.
//
// Requires: each entity's resolvers have distinct Go names (buildResolvers).
// Ensures:  nil, or an error naming both owners of the first Go name that
// two entities' generated resolvers share, or that a per-key batch input
// shares with a schema type.
func checkResolverNames(entities []*Entity, schema *ast.Schema) error {
	types := make(map[string]string, len(schema.Types))
	for name := range schema.Types {
		types[templates.ToGo(name)] = name
	}

	owners := make(map[string]*Entity)
	for _, entity := range entities {
		for _, resolver := range generatedResolvers(entity) {
			goName := templates.ToGo(resolver.name)
			if other, ok := owners[goName]; ok {
				hint := ""
				if !other.PerEntity && !entity.PerEntity && other.Multi != entity.Multi {
					hint = "; giving both the same @entityResolver(multi:) setting avoids this"
				}
				return fmt.Errorf(
					"entities %q and %q both generate entity resolver %s%s",
					other.Name, entity.Name, resolver.name, hint,
				)
			}
			owners[goName] = entity

			if resolver.input == "" {
				continue
			}
			if name, ok := types[templates.ToGo(resolver.input)]; ok {
				return fmt.Errorf(
					"type %q has the same Go name as input %s of entity resolver %s; rename %q",
					name, resolver.input, resolver.name, name,
				)
			}
		}
	}
	return nil
}

// generatedResolver is a field gqlgen generates on the Entity type, with the
// per-key batch input it generates for it, if any.
type generatedResolver struct {
	name  string
	input string
}

// generatedResolvers returns the resolver fields gqlgen generates for
// entity: its one per-entity resolver, or one per @key. A per-entity
// entity's representation inputs are checked in buildRepresentations.
func generatedResolvers(entity *Entity) []generatedResolver {
	if entity.PerEntity {
		return []generatedResolver{{name: entity.PerEntityResolverName()}}
	}
	resolvers := make([]generatedResolver, len(entity.Resolvers))
	for i, resolver := range entity.Resolvers {
		resolvers[i].name = resolver.ResolverName
		if entity.Multi {
			resolvers[i].input = resolver.InputTypeName
		}
	}
	return resolvers
}

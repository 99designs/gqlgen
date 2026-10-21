package federation

import (
	_ "embed"
	"errors"
	"fmt"
	"go/types"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin/federation/fieldset"
)

//go:embed federation_keys.gotpl
var matchedKeysTemplate string

// matchedKeysFilename is the file, beside the generated models, that
// declares the MatchedKey types.
const matchedKeysFilename = "federation_keys.go"

// A per-entity entity's representation reports the @key it matched in its
// MatchedKey field. That field has a named string type, such as
// PortMatchedKey, with one constant per @key holding the key's FieldSet, so a
// resolver that switches on a misspelled or removed key fails to compile.
// They are declared in the model package, in federation_keys.go: the
// representation is declared there, and the federation package, which
// imports the model package, cannot be imported back. They are not a GraphQL
// enum: enum values must be GraphQL names, which would change the strings
// MatchedKey holds.

// matchedKeyTypeName is Entity.MatchedKeyTypeName for the entity named
// entity, for use before its Entity exists.
func matchedKeyTypeName(entity string) string {
	return templates.ToGo(entity) + "MatchedKey"
}

// matchedKeyConstNames names the MatchedKey constants of typeName, one per
// resolver of a per-entity entity, after the Go names keyName gives the
// resolver's key fields: the names readers see on the representation.
//
// Requires: resolvers are one entity's, in @key order.
// Ensures:  names[i] is typeName followed by resolver i's key-field names
// joined with "And", such as PortMatchedKeyIDAndOwnerID; when an earlier key
// already took that name, claimName's suffix makes it distinct.
func matchedKeyConstNames(
	typeName string,
	resolvers []*EntityResolver,
	keyName func(fieldset.Field) string,
) []string {
	names := make([]string, len(resolvers))
	used := make(map[string]bool, len(resolvers))
	for i, resolver := range resolvers {
		fields := make([]string, len(resolver.KeyFields))
		for j, keyField := range resolver.KeyFields {
			fields[j] = keyName(keyField.Field)
		}
		names[i] = claimName(used, typeName+strings.Join(fields, "And"))
	}
	return names
}

// representationMatchedKeyType returns the type of the MatchedKey field of
// the representation struct rep, which modelgen declares as typeName in the
// model package modelPkg.
//
// Ensures: the named type, or an error if rep has no such field or, as when
// someone binds the representation to their own struct, the field has
// another type.
func representationMatchedKeyType(rep types.Type, modelPkg, typeName string) (*types.Named, error) {
	obj, _, _ := types.LookupFieldOrMethod(rep, true, nil, matchedKeyField)
	field, ok := obj.(*types.Var)
	if !ok {
		return nil, fmt.Errorf("representation %s has no %s field", rep, matchedKeyField)
	}
	want := modelPkg + "." + typeName
	keyType, ok := types.Unalias(field.Type()).(*types.Named)
	if !ok || types.TypeString(keyType, nil) != want {
		return nil, fmt.Errorf("%s.%s has type %s; declare it as the generated %s",
			rep, matchedKeyField, field.Type(), want)
	}
	return keyType, nil
}

// modelDecl is a declaration in the model package that has some Go name, with
// the change that frees the name.
type modelDecl struct {
	owner string // what declares it, as errors name it
	fix   string
}

// checkMatchedKeyNames rejects a MatchedKey type or constant whose Go name a
// schema type, a generated enum constant or a representation also has: all
// of them are declared in the model package.
//
// Requires: per-entity entities' resolvers have MatchedKeyConst set
// (buildResolvers); representations come from buildRepresentations.
// Ensures:  nil, or an error naming the first clash, in entity and @key
// order, and how to resolve it.
func checkMatchedKeyNames(
	entities []*Entity,
	schema *ast.Schema,
	representations []*representation,
) error {
	taken := modelDecls(schema, representations)
	for _, entity := range entities {
		if !entity.PerEntity {
			continue
		}
		typeName := entity.MatchedKeyTypeName()
		if other, ok := taken[typeName]; ok {
			return fmt.Errorf("%s has the same Go name as %s, the MatchedKey type of entity %q; %s",
				other.owner, typeName, entity.Name, other.fix)
		}
		for _, resolver := range entity.Resolvers {
			if other, ok := taken[resolver.MatchedKeyConst]; ok {
				return fmt.Errorf(
					"%s has the same Go name as %s, the MatchedKey constant of "+
						"@key(fields: %q) on %q; %s",
					other.owner,
					resolver.MatchedKeyConst,
					resolver.FieldSet(),
					entity.Name,
					other.fix,
				)
			}
		}
	}
	return nil
}

// modelDecls maps the Go names that schema types, generated enum constants
// and representations take in the model package to their declarations. An
// enum constant is named as modelgen names it, enum then value.
func modelDecls(schema *ast.Schema, representations []*representation) map[string]modelDecl {
	taken := make(map[string]modelDecl)
	claim := func(goName string, decl modelDecl) {
		if _, ok := taken[goName]; !ok {
			taken[goName] = decl
		}
	}
	for _, typeName := range slices.Sorted(maps.Keys(schema.Types)) {
		def := schema.Types[typeName]
		claim(templates.ToGo(typeName), modelDecl{
			owner: fmt.Sprintf("type %q", typeName),
			fix:   fmt.Sprintf("rename %q", typeName),
		})
		if def.Kind != ast.Enum {
			continue
		}
		for _, value := range def.EnumValues {
			claim(templates.ToGo(typeName)+templates.ToGo(value.Name), modelDecl{
				owner: fmt.Sprintf("enum value %s.%s", typeName, value.Name),
				fix:   fmt.Sprintf("rename %s.%s", typeName, value.Name),
			})
		}
	}
	for _, r := range representations {
		claim(templates.ToGo(r.Name), modelDecl{
			owner: "representation " + r.Name,
			fix: fmt.Sprintf(
				"give the representation of %q another name in %s",
				r.Owner,
				representationNamesSetting,
			),
		})
	}
	return taken
}

// writeMatchedKeys writes matchedKeysFilename beside cfg's generated models,
// declaring the MatchedKey type and constants of each per-entity entity, or,
// when there is none, removes the file if gqlgen generated it.
//
// It runs in MutateConfig, before modelgen writes the representations that
// refer to these types and reloads the model package.
func writeMatchedKeys(entities []*Entity, cfg *config.Config) error {
	filename := filepath.Join(filepath.Dir(cfg.Model.Filename), matchedKeysFilename)
	perEntity := slices.DeleteFunc(slices.Clone(entities), func(e *Entity) bool {
		return !e.PerEntity
	})
	if len(perEntity) == 0 {
		return removeGenerated(filename)
	}
	return templates.Render(templates.Options{
		PackageName:     cfg.Model.Package,
		Filename:        filename,
		Data:            perEntity,
		GeneratedHeader: true,
		Packages:        cfg.Packages,
		Template:        matchedKeysTemplate,
		PruneOptions:    cfg.GetPruneOptions(),
	})
}

// removeGenerated removes filename if it starts with gqlgen's generated-code
// header, leaving a file someone wrote by hand in place.
//
// Ensures: nil when filename is then absent or not gqlgen's; otherwise the
// error reading or removing it.
func removeGenerated(filename string) error {
	content, err := os.ReadFile(filename)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !strings.HasPrefix(string(content), templates.GeneratedCodeHeader) {
		return nil
	}
	return os.Remove(filename)
}

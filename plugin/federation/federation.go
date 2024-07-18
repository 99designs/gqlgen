package federation

import (
	_ "embed"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/internal/rewrite"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/federation/fieldset"
)

//go:embed federation.gotpl
var federationTemplate string

//go:embed requires.gotpl
var explicitRequiresTemplate string

type federation struct {
	Entities       []*Entity
	Version        int
	PackageOptions PackageOptions
}

type PackageOptions struct {
	// ExplicitRequires will generate a function in the execution context
	// to populate fields using the @required directive into the entity.
	//
	// You can only set one of ExplicitRequires or ComputedRequires to true.
	ExplicitRequires bool
	// ComputedRequires generates resolver functions to compute values for
	// fields using the @required directive.
	ComputedRequires bool
}

// New returns a federation plugin that injects
// federated directives and types into the schema
func New(version int, packageOptions map[string]bool) (plugin.Plugin, error) {
	if version == 0 {
		version = 1
	}

	options, err := buildPackageOptions(packageOptions)
	if err != nil {
		return nil, fmt.Errorf("invalid federation package options: %w", err)
	}
	return &federation{
		Version:        version,
		PackageOptions: options,
	}, nil
}

func buildPackageOptions(packageOptions map[string]bool) (PackageOptions, error) {
	explicitRequires := packageOptions["explicit_requires"]
	computedRequires := packageOptions["computed_requires"]
	if explicitRequires && computedRequires {
		return PackageOptions{}, errors.New("only one of explicit_requires or computed_requires can be set to true")
	}

	return PackageOptions{
		ExplicitRequires: explicitRequires,
		ComputedRequires: computedRequires,
	}, nil
}

// Name returns the plugin name
func (f *federation) Name() string {
	return "federation"
}

// MutateConfig mutates the configuration
func (f *federation) MutateConfig(cfg *config.Config) error {
	builtins := config.TypeMap{
		"_Service": {
			Model: config.StringList{
				"github.com/99designs/gqlgen/plugin/federation/fedruntime.Service",
			},
		},
		"_Entity": {
			Model: config.StringList{
				"github.com/99designs/gqlgen/plugin/federation/fedruntime.Entity",
			},
		},
		"Entity": {
			Model: config.StringList{
				"github.com/99designs/gqlgen/plugin/federation/fedruntime.Entity",
			},
		},
		"_Any": {
			Model: config.StringList{"github.com/99designs/gqlgen/graphql.Map"},
		},
		"federation__Scope": {
			Model: config.StringList{"github.com/99designs/gqlgen/graphql.String"},
		},
		"federation__Policy": {
			Model: config.StringList{"github.com/99designs/gqlgen/graphql.String"},
		},
	}

	for typeName, entry := range builtins {
		if cfg.Models.Exists(typeName) {
			return fmt.Errorf("%v already exists which must be reserved when Federation is enabled", typeName)
		}
		cfg.Models[typeName] = entry
	}
	cfg.Directives["external"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives[directiveRequires] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["provides"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives[directiveKey] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["extends"] = config.DirectiveConfig{SkipRuntime: true}

	// Federation 2 specific directives
	if f.Version == 2 {
		cfg.Directives["shareable"] = config.DirectiveConfig{SkipRuntime: true}
		cfg.Directives["link"] = config.DirectiveConfig{SkipRuntime: true}
		cfg.Directives["tag"] = config.DirectiveConfig{SkipRuntime: true}
		cfg.Directives["override"] = config.DirectiveConfig{SkipRuntime: true}
		cfg.Directives["inaccessible"] = config.DirectiveConfig{SkipRuntime: true}
		cfg.Directives["authenticated"] = config.DirectiveConfig{SkipRuntime: true}
		cfg.Directives["requiresScopes"] = config.DirectiveConfig{SkipRuntime: true}
		cfg.Directives["policy"] = config.DirectiveConfig{SkipRuntime: true}
		cfg.Directives["interfaceObject"] = config.DirectiveConfig{SkipRuntime: true}
		cfg.Directives["composeDirective"] = config.DirectiveConfig{SkipRuntime: true}
	}

	return nil
}

func (f *federation) InjectSourcesEarly() ([]*ast.Source, error) {
	input := ``

	// add version-specific changes on key directive, as well as adding the new directives for federation 2
	if f.Version == 1 {
		input += `
	directive @key(fields: _FieldSet!) repeatable on OBJECT | INTERFACE
	directive @requires(fields: _FieldSet!) on FIELD_DEFINITION
	directive @provides(fields: _FieldSet!) on FIELD_DEFINITION
	directive @extends on OBJECT | INTERFACE
	directive @external on FIELD_DEFINITION
	scalar _Any
	scalar _FieldSet
`
	} else if f.Version == 2 {
		input += `
	directive @authenticated on FIELD_DEFINITION | OBJECT | INTERFACE | SCALAR | ENUM
	directive @composeDirective(name: String!) repeatable on SCHEMA
	directive @extends on OBJECT | INTERFACE
	directive @external on OBJECT | FIELD_DEFINITION
	directive @key(fields: FieldSet!, resolvable: Boolean = true) repeatable on OBJECT | INTERFACE
	directive @inaccessible on
	  | ARGUMENT_DEFINITION
	  | ENUM
	  | ENUM_VALUE
	  | FIELD_DEFINITION
	  | INPUT_FIELD_DEFINITION
	  | INPUT_OBJECT
	  | INTERFACE
	  | OBJECT
	  | SCALAR
	  | UNION
	directive @interfaceObject on OBJECT
	directive @link(import: [String!], url: String!) repeatable on SCHEMA
	directive @override(from: String!, label: String) on FIELD_DEFINITION
	directive @policy(policies: [[federation__Policy!]!]!) on
	  | FIELD_DEFINITION
	  | OBJECT
	  | INTERFACE
	  | SCALAR
	  | ENUM
	directive @provides(fields: FieldSet!) on FIELD_DEFINITION
	directive @requires(fields: FieldSet!) on FIELD_DEFINITION
	directive @requiresScopes(scopes: [[federation__Scope!]!]!) on
	  | FIELD_DEFINITION
	  | OBJECT
	  | INTERFACE
	  | SCALAR
	  | ENUM
	directive @shareable repeatable on FIELD_DEFINITION | OBJECT
	directive @tag(name: String!) repeatable on
	  | ARGUMENT_DEFINITION
	  | ENUM
	  | ENUM_VALUE
	  | FIELD_DEFINITION
	  | INPUT_FIELD_DEFINITION
	  | INPUT_OBJECT
	  | INTERFACE
	  | OBJECT
	  | SCALAR
	  | UNION
	scalar _Any
	scalar FieldSet
	scalar federation__Policy
	scalar federation__Scope
`
	}

	return []*ast.Source{{
		Name:    "federation/directives.graphql",
		Input:   input,
		BuiltIn: true,
	}}, nil
}

// InjectSourceLate creates a GraphQL Entity type with all
// the fields that had the @key directive
func (f *federation) InjectSourcesLate(schema *ast.Schema) ([]*ast.Source, error) {
	f.Entities = buildEntities(schema, f.Version)

	entities := make([]string, 0)
	resolvers := make([]string, 0)
	entityResolverInputDefinitions := make([]string, 0)

	for _, e := range f.Entities {
		if e.Def.Kind != ast.Interface {
			entities = append(entities, e.Name)
		} else if len(schema.GetPossibleTypes(e.Def)) == 0 {
			fmt.Println(
				"skipping @key field on interface " + e.Def.Name + " as no types implement it",
			)
		}

		for _, r := range e.Resolvers {
			// Eventually make this a config. If set to true, we will always use an input type
			// for the resolver rather than individual fields.
			alwaysUseInput := false
			resolverSDL, entityResolverInputSDL := buildResolverSDL(r, e.Multi, alwaysUseInput)
			resolvers = append(resolvers, resolverSDL)
			if entityResolverInputSDL != "" {
				entityResolverInputDefinitions = append(entityResolverInputDefinitions, entityResolverInputSDL)
			}
		}

		if f.PackageOptions.ComputedRequires {
			for _, r := range e.ComputedRequires {
				// We want to force input types for computed fields. The fields selection set
				// can get large and we don't a want a large number of arguments in the resolver.
				alwaysUseInput := true
				resolverSDL, entityResolverInputSDL := buildResolverSDL(r, e.Multi, alwaysUseInput)
				resolvers = append(resolvers, resolverSDL)
				if entityResolverInputSDL != "" {
					entityResolverInputDefinitions = append(entityResolverInputDefinitions, entityResolverInputSDL)
				}
			}
		}
	}

	var blocks []string
	if len(entities) > 0 {
		entitiesSDL := `# a union of all types that use the @key directive
union _Entity = ` + strings.Join(entities, " | ")
		blocks = append(blocks, entitiesSDL)
	}

	// resolvers can be empty if a service defines only "empty
	// extend" types.  This should be rare.
	if len(resolvers) > 0 {
		if len(entityResolverInputDefinitions) > 0 {
			inputSDL := strings.Join(entityResolverInputDefinitions, "\n\n")
			blocks = append(blocks, inputSDL)
		}
		resolversSDL := `# fake type to build resolver interfaces for users to implement
type Entity {
` + strings.Join(resolvers, "\n") + `
}`
		blocks = append(blocks, resolversSDL)
	}

	_serviceTypeDef := `type _Service {
  sdl: String
}`
	blocks = append(blocks, _serviceTypeDef)

	var additionalQueryFields string
	// Quote from the Apollo Federation subgraph specification:
	// If no types are annotated with the key directive, then the
	// _Entity union and _entities field should be removed from the schema
	if len(f.Entities) > 0 {
		additionalQueryFields += `  _entities(representations: [_Any!]!): [_Entity]!
`
	}
	// _service field is required in any case
	additionalQueryFields += `  _service: _Service!`

	extendTypeQueryDef := `extend type ` + schema.Query.Name + ` {
` + additionalQueryFields + `
}`
	blocks = append(blocks, extendTypeQueryDef)

	return []*ast.Source{{
		Name:    "federation/entity.graphql",
		BuiltIn: true,
		Input:   "\n" + strings.Join(blocks, "\n\n") + "\n",
	}}, nil
}

func (f *federation) GenerateCode(data *codegen.Data) error {
	// requires imports
	requiresImports := make(map[string]bool, 0)
	requiresImports["context"] = true
	requiresImports["fmt"] = true

	requiresEntities := make(map[string]*Entity, 0)

	// Save package options on f for template use
	packageOptions, err := buildPackageOptions(data.Config.Federation.Options)
	if err != nil {
		return fmt.Errorf("invalid federation package options: %w", err)
	}
	f.PackageOptions = packageOptions

	if len(f.Entities) > 0 {
		if data.Objects.ByName("Entity") != nil {
			data.Objects.ByName("Entity").Root = true
		}
		for _, e := range f.Entities {
			obj := data.Objects.ByName(e.Def.Name)

			if e.Def.Kind == ast.Interface {
				if len(data.Interfaces[e.Def.Name].Implementors) == 0 {
					fmt.Println(
						"skipping @key field on interface " + e.Def.Name + " as no types implement it",
					)
					continue
				}
				obj = data.Objects.ByName(data.Interfaces[e.Def.Name].Implementors[0].Name)
			}

			for _, r := range e.Resolvers {
				populateKeyFieldTypes(r, obj, data.Objects, e.Def.Name)
			}

			for _, r := range e.ComputedRequires {
				populateKeyFieldTypes(r, obj, data.Objects, e.Def.Name)
			}

			// fill in types for requires fields
			//
			for _, reqField := range e.Requires {
				if len(reqField.Field) == 0 {
					fmt.Println("skipping @requires field " + reqField.Name + " in " + e.Def.Name)
					continue
				}
				// keep track of which entities have requires
				requiresEntities[e.Def.Name] = e
				// make a proper import path
				typeString := strings.Split(obj.Type.String(), ".")
				requiresImports[strings.Join(typeString[:len(typeString)-1], ".")] = true

				cgField := reqField.Field.TypeReference(obj, data.Objects)
				reqField.Type = cgField.TypeReference
			}

			// add type info to entity
			e.Type = obj.Type
		}
	}

	// fill in types for resolver inputs
	//
	for _, entity := range f.Entities {
		if f.PackageOptions.ComputedRequires {
			for _, resolver := range entity.ComputedRequires {
				obj := data.Inputs.ByName(resolver.InputTypeName)
				if obj == nil {
					return fmt.Errorf("input object %s not found", resolver.InputTypeName)
				}

				resolver.InputType = obj.Type
			}
		}

		if !entity.Multi {
			continue
		}

		for _, resolver := range entity.Resolvers {
			obj := data.Inputs.ByName(resolver.InputTypeName)
			if obj == nil {
				return fmt.Errorf("input object %s not found", resolver.InputTypeName)
			}

			resolver.InputType = obj.Type
		}
	}

	if f.PackageOptions.ExplicitRequires && len(requiresEntities) > 0 {
		// check for existing requires functions
		type Populator struct {
			FuncName       string
			Exists         bool
			Comment        string
			Implementation string
			Entity         *Entity
		}
		populators := make([]Populator, 0)

		rewriter, err := rewrite.New(data.Config.Federation.Dir())
		if err != nil {
			return err
		}

		for name, entity := range requiresEntities {
			populator := Populator{
				FuncName: fmt.Sprintf("Populate%sRequires", name),
				Entity:   entity,
			}

			populator.Comment = strings.TrimSpace(strings.TrimLeft(rewriter.GetMethodComment("executionContext", populator.FuncName), `\`))
			populator.Implementation = strings.TrimSpace(rewriter.GetMethodBody("executionContext", populator.FuncName))

			if populator.Implementation == "" {
				populator.Exists = false
				populator.Implementation = fmt.Sprintf("panic(fmt.Errorf(\"not implemented: %v\"))", populator.FuncName)
			}
			populators = append(populators, populator)
		}

		sort.Slice(populators, func(i, j int) bool {
			return populators[i].FuncName < populators[j].FuncName
		})

		requiresFile := data.Config.Federation.Dir() + "/federation.requires.go"
		existingImports := rewriter.ExistingImports(requiresFile)
		for _, imp := range existingImports {
			if imp.Alias == "" {
				// import exists in both places, remove
				delete(requiresImports, imp.ImportPath)
			}
		}

		for k := range requiresImports {
			existingImports = append(existingImports, rewrite.Import{ImportPath: k})
		}

		// render requires populators
		err = templates.Render(templates.Options{
			PackageName: data.Config.Federation.Package,
			Filename:    requiresFile,
			Data: struct {
				federation
				ExistingImports []rewrite.Import
				Populators      []Populator
				OriginalSource  string
			}{*f, existingImports, populators, ""},
			GeneratedHeader: false,
			Packages:        data.Config.Packages,
			Template:        explicitRequiresTemplate,
		})
		if err != nil {
			return err
		}
	}

	return templates.Render(templates.Options{
		PackageName: data.Config.Federation.Package,
		Filename:    data.Config.Federation.Filename,
		Data: struct {
			federation
			UsePointers bool
		}{*f, data.Config.ResolversAlwaysReturnPointers},
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
		Template:        federationTemplate,
	})
}

// Fill in types for key fields
func populateKeyFieldTypes(
	resolver *EntityResolver,
	obj *codegen.Object,
	allObjects codegen.Objects,
	name string,
) {
	for _, keyField := range resolver.KeyFields {
		if len(keyField.Field) == 0 {
			fmt.Println(
				"skipping @key field " + keyField.Definition.Name + " in " + resolver.ResolverName + " in " + name,
			)
			continue
		}
		cgField := keyField.Field.TypeReference(obj, allObjects)
		keyField.Type = cgField.TypeReference
	}
}

func buildEntities(schema *ast.Schema, version int) []*Entity {
	entities := make([]*Entity, 0)
	for _, schemaType := range schema.Types {
		entity := buildEntity(schemaType, schema, version)
		if entity != nil {
			entities = append(entities, entity)
		}
	}

	// make sure order remains stable across multiple builds
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].Name < entities[j].Name
	})

	return entities
}

func buildEntity(
	schemaType *ast.Definition,
	schema *ast.Schema,
	version int,
) *Entity {
	keys, ok := isFederatedEntity(schemaType)
	if !ok {
		return nil
	}

	if (schemaType.Kind == ast.Interface) && (len(schema.GetPossibleTypes(schemaType)) == 0) {
		fmt.Printf("@key directive found on unused \"interface %s\". Will be ignored.\n", schemaType.Name)
		return nil
	}

	entity := &Entity{
		Name:      schemaType.Name,
		Def:       schemaType,
		Resolvers: nil,
		Requires:  nil,
		Multi:     isMultiEntity(schemaType),
	}

	// If our schema has a field with a type defined in
	// another service, then we need to define an "empty
	// extend" of that type in this service, so this service
	// knows what the type is like.  But the graphql-server
	// will never ask us to actually resolve this "empty
	// extend", so we don't require a resolver function for
	// it.  (Well, it will never ask in practice; it's
	// unclear whether the spec guarantees this.  See
	// https://github.com/apollographql/apollo-server/issues/3852
	// ).  Example:
	//    type MyType {
	//       myvar: TypeDefinedInOtherService
	//    }
	//    // Federation needs this type, but
	//    // it doesn't need a resolver for it!
	//    extend TypeDefinedInOtherService @key(fields: "id") {
	//       id: ID @external
	//    }
	if entity.allFieldsAreExternal(version) {
		return entity
	}

	entity.Resolvers = buildResolvers(schemaType, schema, keys, entity.Multi)
	entity.ComputedRequires = buildComputedRequires(schemaType, schema, entity.Multi)
	entity.Requires = buildRequires(schemaType)

	return entity
}

func isMultiEntity(schemaType *ast.Definition) bool {
	dir := schemaType.Directives.ForName("entityResolver")
	if dir == nil {
		return false
	}

	if dirArg := dir.Arguments.ForName("multi"); dirArg != nil {
		if dirVal, err := dirArg.Value.Value(nil); err == nil {
			return dirVal.(bool)
		}
	}

	return false
}

func buildResolvers(
	schemaType *ast.Definition,
	schema *ast.Schema,
	keys []*ast.Directive,
	multi bool,
) []*EntityResolver {
	resolvers := make([]*EntityResolver, 0)
	for _, dir := range keys {
		if len(dir.Arguments) > 2 {
			panic("More than two arguments provided for @key declaration.")
		}
		keyFields, resolverFields := buildKeyFields(
			schemaType,
			schema,
			dir,
		)

		resolverFieldsToGo := schemaType.Name + "By" + strings.Join(resolverFields, "And")
		var resolverName string
		if multi {
			resolverFieldsToGo += "s" // Pluralize for better API readability
			resolverName = fmt.Sprintf("findMany%s", resolverFieldsToGo)
		} else {
			resolverName = fmt.Sprintf("find%s", resolverFieldsToGo)
		}

		resolvers = append(resolvers, &EntityResolver{
			ResolverName:   resolverName,
			KeyFields:      keyFields,
			InputTypeName:  resolverFieldsToGo + "Input",
			ReturnTypeName: schemaType.Name,
			NonNull:        true,
		})
	}

	return resolvers
}

func buildComputedRequires(
	schemaType *ast.Definition,
	schema *ast.Schema,
	multi bool,
) []*EntityResolver {
	resolvers := make([]*EntityResolver, 0)
	for _, f := range schemaType.Fields {
		dir := f.Directives.ForName(directiveRequires)
		if dir == nil {
			continue
		}

		resolver := buildComputedRequiresForField(schemaType, schema, f, dir, multi)
		if resolver != nil {
			resolvers = append(resolvers, resolver)
		}
	}

	return resolvers
}

func buildComputedRequiresForField(
	schemaType *ast.Definition,
	schema *ast.Schema,
	field *ast.FieldDefinition,
	dir *ast.Directive,
	multi bool,
) *EntityResolver {
	keyFields, resolverFields := buildKeyFields(
		schemaType,
		schema,
		dir,
	)

	resolverFieldsToGo := schemaType.Name + "With" + strings.Join(resolverFields, "And")
	var resolverName string
	if multi {
		resolverFieldsToGo += "s" // Pluralize for better API readability
		resolverName = fmt.Sprintf("computeMany%s", resolverFieldsToGo)
	} else {
		resolverName = fmt.Sprintf("compute%s", resolverFieldsToGo)
	}

	return &EntityResolver{
		ResolverName:   resolverName,
		KeyFields:      keyFields,
		InputTypeName:  "Compute" + resolverFieldsToGo + "Input",
		ReturnTypeName: field.Type.Name(),
		NonNull:        field.Type.NonNull,
		FieldName:      field.Name,
	}
}

func buildKeyFields(
	schemaType *ast.Definition,
	schema *ast.Schema,
	dir *ast.Directive,
) ([]*KeyField, []string) {
	var arg *ast.Argument

	// since directives are able to now have multiple arguments, we need to check both possible for a possible @key(fields="" fields="")
	for _, a := range dir.Arguments {
		if a.Name == directiveArgFields {
			if arg != nil {
				panic("More than one `fields` provided for @requires declaration.")
			}
			arg = a
		}
	}

	keyFieldSet := fieldset.New(arg.Value.Raw, nil)

	keyFields := make([]*KeyField, len(keyFieldSet))
	resolverFields := []string{}
	for i, field := range keyFieldSet {
		def := field.FieldDefinition(schemaType, schema)

		if def == nil {
			panic(fmt.Sprintf("no field for %v", field))
		}

		keyFields[i] = &KeyField{Definition: def, Field: field}
		resolverFields = append(resolverFields, keyFields[i].Field.ToGo())
	}

	return keyFields, resolverFields
}

func buildRequires(schemaType *ast.Definition) []*Requires {
	requires := make([]*Requires, 0)
	for _, f := range schemaType.Fields {
		dir := f.Directives.ForName(directiveRequires)
		if dir == nil {
			continue
		}
		if len(dir.Arguments) != 1 || dir.Arguments[0].Name != directiveArgFields {
			panic("Exactly one `fields` argument needed for @requires declaration.")
		}
		requiresFieldSet := fieldset.New(dir.Arguments[0].Value.Raw, nil)
		for _, field := range requiresFieldSet {
			requires = append(requires, &Requires{
				Name:  field.ToGoPrivate(),
				Field: field,
			})
		}
	}

	return requires
}

func isFederatedEntity(schemaType *ast.Definition) ([]*ast.Directive, bool) {
	switch schemaType.Kind {
	case ast.Object:
		keys := schemaType.Directives.ForNames(directiveKey)
		if len(keys) > 0 {
			return keys, true
		}
	case ast.Interface:
		keys := schemaType.Directives.ForNames(directiveKey)
		if len(keys) > 0 {
			return keys, true
		}

		// TODO: support @extends for interfaces
		if dir := schemaType.Directives.ForName("extends"); dir != nil {
			panic(
				fmt.Sprintf(
					"@extends directive is not currently supported for interfaces, use \"extend interface %s\" instead.",
					schemaType.Name,
				))
		}
	default:
		// ignore
	}
	return nil, false
}

func buildResolverSDL(
	resolver *EntityResolver,
	multi bool,
	alwaysUseInput bool,
) (resolverSDL string, entityResolverInputSDL string) {
	if multi {
		entityResolverInputSDL = buildEntityResolverInputDefinitionSDL(resolver)
		resolverSDL := fmt.Sprintf("\t%s(reps: [%s]!): [%s]", resolver.ResolverName, resolver.InputTypeName, resolver.ReturnTypeName)
		return resolverSDL, entityResolverInputSDL
	}

	if alwaysUseInput {
		entityResolverInputSDL = buildEntityResolverInputDefinitionSDL(resolver)
		resolverSDL := fmt.Sprintf("\t%s(reps: %s): %s", resolver.ResolverName, resolver.InputTypeName, resolver.ReturnTypeName)
		if resolver.NonNull {
			resolverSDL += "!"
		}
		return resolverSDL, entityResolverInputSDL
	}

	resolverArgs := ""
	for _, keyField := range resolver.KeyFields {
		resolverArgs += fmt.Sprintf("%s: %s,", keyField.Field.ToGoPrivate(), keyField.Definition.Type.String())
	}
	resolverSDL = fmt.Sprintf("\t%s(%s): %s", resolver.ResolverName, resolverArgs, resolver.ReturnTypeName)
	if resolver.NonNull {
		resolverSDL += "!"
	}
	return resolverSDL, ""
}

func buildEntityResolverInputDefinitionSDL(resolver *EntityResolver) string {
	entityResolverInputDefinition := "input " + resolver.InputTypeName + " {\n"
	for _, keyField := range resolver.KeyFields {
		entityResolverInputDefinition += fmt.Sprintf(
			"\t%s: %s\n",
			keyField.Field.ToGo(),
			keyField.Definition.Type.String(),
		)
	}
	return entityResolverInputDefinition + "}"
}

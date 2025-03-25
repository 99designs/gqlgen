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
	"github.com/99designs/gqlgen/plugin/federation/fieldset"
)

//go:embed federation.gotpl
var federationTemplate string

//go:embed requires.gotpl
var explicitRequiresTemplate string

type Federation struct {
	Entities       []*Entity
	PackageOptions PackageOptions

	version int

	// true if @requires is used in the schema
	usesRequires bool
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
func New(version int, cfg *config.Config) (*Federation, error) {
	if version == 0 {
		version = 1
	}

	options, err := buildPackageOptions(cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid federation package options: %w", err)
	}
	return &Federation{
		version:        version,
		PackageOptions: options,
	}, nil
}

func buildPackageOptions(cfg *config.Config) (PackageOptions, error) {
	packageOptions := cfg.Federation.Options

	explicitRequires := packageOptions["explicit_requires"]
	computedRequires := packageOptions["computed_requires"]
	if explicitRequires && computedRequires {
		return PackageOptions{}, errors.New("only one of explicit_requires or computed_requires can be set to true")
	}

	if computedRequires {
		if cfg.Federation.Version != 2 {
			return PackageOptions{}, errors.New("when using federation.options.computed_requires you must be using Federation 2")
		}

		// We rely on injecting a null argument with a directives for fields with @requires, so we need to ensure
		// our directive is always called.
		if !cfg.CallArgumentDirectivesWithNull {
			return PackageOptions{}, errors.New("when using federation.options.computed_requires, call_argument_directives_with_null must be set to true")
		}
	}

	// We rely on injecting a null argument with a directives for fields with @requires, so we need to ensure
	// our directive is always called.

	return PackageOptions{
		ExplicitRequires: explicitRequires,
		ComputedRequires: computedRequires,
	}, nil
}

// Name returns the plugin name
func (f *Federation) Name() string {
	return "federation"
}

// MutateConfig mutates the configuration
func (f *Federation) MutateConfig(cfg *config.Config) error {
	for typeName, entry := range builtins {
		if cfg.Models.Exists(typeName) {
			return fmt.Errorf("%v already exists which must be reserved when Federation is enabled", typeName)
		}
		cfg.Models[typeName] = entry
	}
	cfg.Directives["external"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives[dirNameRequires] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["provides"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives[dirNameKey] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["extends"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives[dirNameEntityResolver] = config.DirectiveConfig{SkipRuntime: true}

	// Federation 2 specific directives
	if f.version == 2 {
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

	if f.usesRequires && f.PackageOptions.ComputedRequires {
		cfg.Schema.Directives[dirPopulateFromRepresentations.Name] = dirPopulateFromRepresentations
		cfg.Directives[dirPopulateFromRepresentations.Name] = config.DirectiveConfig{Implementation: &populateFromRepresentationsImplementation}

		cfg.Schema.Directives[dirEntityReference.Name] = dirEntityReference
		cfg.Directives[dirEntityReference.Name] = config.DirectiveConfig{SkipRuntime: true}

		f.addMapType(cfg)
		f.mutateSchemaForRequires(cfg.Schema, cfg)
	}

	return nil
}

func (f *Federation) InjectSourcesEarly() ([]*ast.Source, error) {
	input := ``

	// add version-specific changes on key directive, as well as adding the new directives for federation 2
	switch f.version {
	case 1:
		input += federationVersion1Schema
	case 2:
		input += federationVersion2Schema
	}

	return []*ast.Source{{
		Name:    dirGraphQLQFile,
		Input:   input,
		BuiltIn: true,
	}}, nil
}

// InjectSourceLate creates a GraphQL Entity type with all
// the fields that had the @key directive
func (f *Federation) InjectSourcesLate(schema *ast.Schema) ([]*ast.Source, error) {
	f.Entities = f.buildEntities(schema, f.version)

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
			resolverSDL, entityResolverInputSDL := buildResolverSDL(r, e.Multi)
			resolvers = append(resolvers, resolverSDL)
			if entityResolverInputSDL != "" {
				entityResolverInputDefinitions = append(entityResolverInputDefinitions, entityResolverInputSDL)
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
		Name:    entityGraphQLQFile,
		BuiltIn: true,
		Input:   "\n" + strings.Join(blocks, "\n\n") + "\n",
	}}, nil
}

func (f *Federation) GenerateCode(data *codegen.Data) error {
	// requires imports
	requiresImports := make(map[string]bool, 0)
	requiresImports["context"] = true
	requiresImports["fmt"] = true

	requiresEntities := make(map[string]*Entity, 0)

	// Save package options on f for template use
	packageOptions, err := buildPackageOptions(data.Config)
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

				if containsUnionField(reqField) {
					continue
				}

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
		err := f.generateExplicitRequires(
			data,
			requiresEntities,
			requiresImports,
		)
		if err != nil {
			return err
		}
	}

	return templates.Render(templates.Options{
		PackageName: data.Config.Federation.Package,
		Filename:    data.Config.Federation.Filename,
		Data: struct {
			Federation
			UsePointers                          bool
			UseFunctionSyntaxForExecutionContext bool
		}{*f, data.Config.ResolversAlwaysReturnPointers, data.Config.UseFunctionSyntaxForExecutionContext},
		GeneratedHeader: true,
		Packages:        data.Config.Packages,
		Template:        federationTemplate,
	})
}

func containsUnionField(reqField *Requires) bool {
	for _, requireFields := range reqField.Field {
		if strings.HasPrefix(requireFields, "... on") {
			return true
		}
	}
	return false
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

func (f *Federation) buildEntities(schema *ast.Schema, version int) []*Entity {
	entities := make([]*Entity, 0)
	for _, schemaType := range schema.Types {
		entity := f.buildEntity(schemaType, schema, version)
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

func (f *Federation) buildEntity(
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
	entity.Requires = buildRequires(schemaType)
	if len(entity.Requires) > 0 {
		f.usesRequires = true
	}

	return entity
}

func isMultiEntity(schemaType *ast.Definition) bool {
	dir := schemaType.Directives.ForName(dirNameEntityResolver)
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
		})
	}

	return resolvers
}

func extractFields(
	dir *ast.Directive,
) (string, error) {
	var arg *ast.Argument

	// since directives are able to now have multiple arguments, we need to check both possible for a possible @key(fields="" fields="")
	for _, a := range dir.Arguments {
		if a.Name == DirArgFields {
			if arg != nil {
				return "", errors.New("more than one \"fields\" argument provided for declaration")
			}
			arg = a
		}
	}

	return arg.Value.Raw, nil
}

func buildKeyFields(
	schemaType *ast.Definition,
	schema *ast.Schema,
	dir *ast.Directive,
) ([]*KeyField, []string) {
	fieldsRaw, err := extractFields(dir)
	if err != nil {
		panic("More than one `fields` argument provided for declaration.")
	}

	keyFieldSet := fieldset.New(fieldsRaw, nil)

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
		dir := f.Directives.ForName(dirNameRequires)
		if dir == nil {
			continue
		}

		fieldsRaw, err := extractFields(dir)
		if err != nil {
			panic("Exactly one `fields` argument needed for @requires declaration.")
		}
		requiresFieldSet := fieldset.New(fieldsRaw, nil)
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
		keys := schemaType.Directives.ForNames(dirNameKey)
		if len(keys) > 0 {
			return keys, true
		}
	case ast.Interface:
		keys := schemaType.Directives.ForNames(dirNameKey)
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

func (f *Federation) generateExplicitRequires(
	data *codegen.Data,
	requiresEntities map[string]*Entity,
	requiresImports map[string]bool,
) error {
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
	return templates.Render(templates.Options{
		PackageName: data.Config.Federation.Package,
		Filename:    requiresFile,
		Data: struct {
			Federation
			ExistingImports []rewrite.Import
			Populators      []Populator
			OriginalSource  string
		}{*f, existingImports, populators, ""},
		GeneratedHeader: false,
		Packages:        data.Config.Packages,
		Template:        explicitRequiresTemplate,
	})
}

func buildResolverSDL(
	resolver *EntityResolver,
	multi bool,
) (resolverSDL, entityResolverInputSDL string) {
	if multi {
		entityResolverInputSDL = buildEntityResolverInputDefinitionSDL(resolver)
		resolverSDL := fmt.Sprintf("\t%s(reps: [%s]!): [%s]", resolver.ResolverName, resolver.InputTypeName, resolver.ReturnTypeName)
		return resolverSDL, entityResolverInputSDL
	}

	resolverArgs := ""
	for _, keyField := range resolver.KeyFields {
		resolverArgs += fmt.Sprintf("%s: %s,", keyField.Field.ToGoPrivate(), keyField.Definition.Type.String())
	}
	resolverSDL = fmt.Sprintf("\t%s(%s): %s!", resolver.ResolverName, resolverArgs, resolver.ReturnTypeName)
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

func (f *Federation) addMapType(cfg *config.Config) {
	cfg.Models[mapTypeName] = config.TypeMapEntry{
		Model: config.StringList{"github.com/99designs/gqlgen/graphql.Map"},
	}
	cfg.Schema.Types[mapTypeName] = &ast.Definition{
		Kind:        ast.Scalar,
		Name:        mapTypeName,
		Description: "Maps an arbitrary GraphQL value to a map[string]any Go type.",
	}
}

func (f *Federation) mutateSchemaForRequires(
	schema *ast.Schema,
	cfg *config.Config,
) {
	for _, schemaType := range schema.Types {
		for _, field := range schemaType.Fields {
			if dir := field.Directives.ForName(dirNameRequires); dir != nil {
				// ensure we always generate a resolver for any @requires field
				model := cfg.Models[schemaType.Name]
				fieldConfig := model.Fields[field.Name]
				fieldConfig.Resolver = true
				if model.Fields == nil {
					model.Fields = make(map[string]config.TypeMapField)
				}
				model.Fields[field.Name] = fieldConfig
				cfg.Models[schemaType.Name] = model

				requiresArgument := &ast.ArgumentDefinition{
					Name: fieldArgRequires,
					Type: ast.NamedType(mapTypeName, nil),
					Directives: ast.DirectiveList{
						{
							Name:       dirNamePopulateFromRepresentations,
							Definition: dirPopulateFromRepresentations,
						},
					},
				}
				field.Arguments = append(field.Arguments, requiresArgument)
			}
		}
	}
}

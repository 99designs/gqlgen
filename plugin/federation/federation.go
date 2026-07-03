package federation

import (
	_ "embed"
	"errors"
	"fmt"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

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
	Entities         []*Entity
	RequiresEntities map[string]*Entity
	PackageOptions   PackageOptions

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
	// EntityResolverMulti is default engine for entityResolver generation.
	// This can be overriding by @entityResolver(multi: Boolean) directive.
	// false by default.
	EntityResolverMulti bool
	// PreloadedRequires sets "preloaded" as the package-level default @requires
	// strategy; individual entities may override it via
	// @entityResolver(requires: "…"). Under preloaded, the generated multi
	// resolver input (<Entity>By<Keys>sInput) carries the entity's @requires
	// fields, unmarshaled before the resolver runs, so a multi resolver sees
	// every entity's @requires data in one scope — enabling a naturally-batched
	// computation (e.g. one ML-inference call across the whole batch). Multi
	// entities only. false by default.
	PreloadedRequires bool
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

// federation.options keys, recognized in buildPackageOptions and referenced in
// user-facing errors, so their spelling has a single source of truth.
const (
	optionExplicitRequires    = "explicit_requires"
	optionComputedRequires    = "computed_requires"
	optionEntityResolverMulti = "entity_resolver_multi"
	optionPreloadedRequires   = "preloaded_requires"
)

func buildPackageOptions(cfg *config.Config) (PackageOptions, error) {
	packageOptions := cfg.Federation.Options

	var explicitRequires,
		computedRequires,
		entityResolverMulti,
		preloadedRequires bool

	for k, v := range packageOptions {
		switch k {
		case optionExplicitRequires:
			explicitRequires = v
		case optionComputedRequires:
			computedRequires = v
		case optionEntityResolverMulti:
			entityResolverMulti = v
		case optionPreloadedRequires:
			preloadedRequires = v
		default:
			return PackageOptions{}, fmt.Errorf("unknown package option: %s", k)
		}
	}

	if explicitRequires && computedRequires {
		return PackageOptions{}, fmt.Errorf(
			"only one of %s or %s can be set to true",
			optionExplicitRequires,
			optionComputedRequires,
		)
	}

	// The preloaded strategy delivers @requires data on the representation passed
	// *into* the batch resolver, so it owns @requires handling itself. The other
	// two @requires strategies are therefore incompatible with it:
	//   - computed_requires resolves @requires as separate field resolvers, so
	//     they would never be present on the representation; and
	//   - explicit_requires generates a Populate<Entity>Requires stub that
	//     writes @requires onto the *returned* entity, which preloaded
	//     never calls — silently ignoring the user's populator.
	// Reject both combinations rather than generate something that looks wired
	// up but drops the @requires data.
	if preloadedRequires && computedRequires {
		return PackageOptions{}, fmt.Errorf(
			"%s cannot be combined with %s: computed requires are resolved as separate field resolvers, so they are not available on the representation passed to the batch resolver",
			optionPreloadedRequires,
			optionComputedRequires,
		)
	}
	if preloadedRequires && explicitRequires {
		return PackageOptions{}, fmt.Errorf(
			"%s cannot be combined with %s: preloaded populates @requires on the representation passed to the batch resolver, so the Populate<Entity>Requires stub would never run",
			optionPreloadedRequires,
			optionExplicitRequires,
		)
	}

	// The same prerequisites are re-checked per entity in MutateConfig (computed
	// can also be selected by the @entityResolver(requires:) directive without
	// the package option); failing here too keeps New() validating the package
	// option up front.
	if computedRequires {
		if err := computedRequiresPrerequisiteError(
			cfg.Federation.Version,
			cfg.CallArgumentDirectivesWithNull,
		); err != nil {
			return PackageOptions{}, err
		}
	}

	// We rely on injecting a null argument with a directives for fields with @requires, so we need
	// to ensure
	// our directive is always called.
	return PackageOptions{
		ExplicitRequires:    explicitRequires,
		ComputedRequires:    computedRequires,
		EntityResolverMulti: entityResolverMulti,
		PreloadedRequires:   preloadedRequires,
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
			return fmt.Errorf(
				"%v already exists which must be reserved when Federation is enabled",
				typeName,
			)
		}
		cfg.Models[typeName] = entry
	}
	cfg.Directives["external"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives[dirNameRequires] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["provides"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives[dirNameKey] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives["extends"] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives[dirNameEntityResolver] = config.DirectiveConfig{SkipRuntime: true}
	cfg.Directives[dirNameComputedRequires] = config.DirectiveConfig{SkipRuntime: true}

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

	if f.anyComputedRequiresField() {
		if err := computedRequiresPrerequisiteError(
			f.version,
			cfg.CallArgumentDirectivesWithNull,
		); err != nil {
			return err
		}

		cfg.Schema.Directives[dirPopulateFromRepresentations.Name] = dirPopulateFromRepresentations
		cfg.Directives[dirPopulateFromRepresentations.Name] = config.DirectiveConfig{
			Implementation: &populateFromRepresentationsImplementation,
		}

		cfg.Schema.Directives[dirEntityReference.Name] = dirEntityReference
		cfg.Directives[dirEntityReference.Name] = config.DirectiveConfig{SkipRuntime: true}

		f.addMapType(cfg)
		f.mutateSchemaForRequires(cfg)
	}

	if f.anyEntityWithStrategy(RequiresPreloaded) {
		if err := f.injectPreloadedRequiresFields(cfg); err != nil {
			return err
		}
	}

	return nil
}

// anyEntityWithStrategy reports whether any built entity that has @requires
// fields resolves to the given strategy.
func (f *Federation) anyEntityWithStrategy(strategy RequiresStrategy) bool {
	for _, e := range f.Entities {
		if len(e.Requires) > 0 && e.RequiresStrategy == strategy {
			return true
		}
	}
	return false
}

// anyComputedRequiresField reports whether any @requires field on any built
// entity is computed (via @computedRequires or the computed_requires package option).
// The computed strategy is per field, so its schema mutation and prerequisites
// are gated on this rather than on a whole-entity strategy.
func (f *Federation) anyComputedRequiresField() bool {
	for _, e := range f.Entities {
		for _, req := range e.Requires {
			if req.Computed {
				return true
			}
		}
	}
	return false
}

// computedRequiresPrerequisiteError returns a non-nil error when the computed
// @requires strategy is used without its prerequisites: Federation 2 and
// call_argument_directives_with_null (it relies on injecting a null directive
// argument that must always be called).
func computedRequiresPrerequisiteError(version int, callArgumentDirectivesWithNull bool) error {
	if version != 2 {
		return errors.New("when using computed @requires you must be using Federation 2")
	}
	if !callArgumentDirectivesWithNull {
		return errors.New(
			"when using computed @requires, call_argument_directives_with_null must be set to true",
		)
	}
	return nil
}

// injectPreloadedRequiresFields augments each multi entity's generated
// representation input type with that entity's @requires fields, as modelgen
// ExtraFields. modelgen runs after the federation plugin in the MutateConfig
// phase, so the fields land on the generated struct; the result is a resolver
// signature of findManyX(ctx, []*<Entity>By<Keys>sInput) where the input now
// carries both @key and @requires fields. The federation template then
// populates them before calling the resolver, giving the batch resolver a
// single scope with every entity's @requires data.
//
// Two restrictions are enforced here rather than left to fail in generated
// code:
//
//   - Flat (single-segment) @requires only. A nested @requires such as
//     "world { foo }" would require allocating intermediate objects on the
//     representation before assigning the leaf.
//   - Scalar/enum @requires only. Output object types (e.g. a required
//     `variations: [Variation!]!`) have no unmarshaler — gqlgen can only
//     unmarshal scalar leaves of a representation — so a required object field
//     cannot be reconstructed onto the representation. This is a pre-existing
//     gqlgen limitation, not specific to preloaded; the README case
//     study works around it by requiring the scalar leaves (`variations { price
//     imageUrl id }`), which is the nested form excluded above.
func (f *Federation) injectPreloadedRequiresFields(cfg *config.Config) error {
	binder := cfg.NewBinder()
	for _, entity := range f.Entities {
		if !entity.IsPreloaded() || len(entity.Requires) == 0 {
			continue
		}
		for _, req := range entity.Requires {
			// Computed @requires fields are delivered by a field resolver, not
			// populated onto the resolver input, so they are not ExtraFields and
			// are exempt from the scalar-only restriction below.
			if req.Computed {
				continue
			}
			fieldDef, err := preloadedRequiresField(cfg, entity, req)
			if err != nil {
				return err
			}
			ref, err := binder.TypeReference(fieldDef.Type, nil)
			if err != nil {
				return fmt.Errorf(
					"resolving Go type for @requires field %q on entity %q: %w",
					req.Field[0],
					entity.Def.Name,
					err,
				)
			}
			goType := types.TypeString(ref.GO, func(p *types.Package) string {
				if p == nil {
					return ""
				}
				return p.Path()
			})
			for _, resolver := range entity.Resolvers {
				model := cfg.Models[resolver.InputTypeName]
				if model.ExtraFields == nil {
					model.ExtraFields = make(map[string]config.ModelExtraField)
				}
				model.ExtraFields[req.Field.ToGo()] = config.ModelExtraField{Type: goType}
				cfg.Models[resolver.InputTypeName] = model
			}
		}
	}
	return nil
}

// preloadedRequiresField validates that a single @requires entry is
// supported under preloaded and returns its field definition.
func preloadedRequiresField(
	cfg *config.Config,
	entity *Entity,
	req *Requires,
) (*ast.FieldDefinition, error) {
	if len(req.Field) != 1 {
		return nil, fmt.Errorf(
			"preloaded_requires does not support nested @requires field %q on entity %q; only flat @requires fields are supported",
			req.Field.Join("."),
			entity.Def.Name,
		)
	}
	fieldDef := entity.Def.Fields.ForName(req.Field[0])
	if fieldDef == nil {
		return nil, fmt.Errorf(
			"@requires field %q not found on entity %q",
			req.Field[0],
			entity.Def.Name,
		)
	}
	baseName := fieldDef.Type
	for baseName.Elem != nil {
		baseName = baseName.Elem
	}
	if def := cfg.Schema.Types[baseName.NamedType]; def != nil &&
		def.Kind != ast.Scalar && def.Kind != ast.Enum {
		return nil, fmt.Errorf(
			"preloaded_requires does not support @requires field %q on entity %q: only scalar and enum fields can be reconstructed onto the representation, not %s %q",
			req.Field[0],
			entity.Def.Name,
			def.Kind,
			baseName.NamedType,
		)
	}
	return fieldDef, nil
}

func (f *Federation) InjectSourcesEarly() ([]*ast.Source, error) {
	input := ``

	// add version-specific changes on key directive, as well as adding the new directives for
	// federation 2
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

// InjectSourcesLate creates a GraphQL Entity type with all
// the fields that had the @key directive
func (f *Federation) InjectSourcesLate(schema *ast.Schema) ([]*ast.Source, error) {
	builtEntities, err := f.buildEntities(schema, f.version)
	if err != nil {
		return nil, err
	}
	f.Entities = builtEntities

	entityNames := make([]string, 0)
	resolvers := make([]string, 0)
	entityResolverInputDefinitions := make([]string, 0)
	for _, e := range f.Entities {
		if e.Def.Kind != ast.Interface {
			entityNames = append(entityNames, e.Name)
		} else if len(schema.GetPossibleTypes(e.Def)) == 0 {
			fmt.Println(
				"skipping @key field on interface " + e.Def.Name + " as no types implement it",
			)
		}

		for _, r := range e.Resolvers {
			resolverSDL, entityResolverInputSDL := buildResolverSDL(r, e.Multi)
			resolvers = append(resolvers, resolverSDL)
			if entityResolverInputSDL != "" {
				entityResolverInputDefinitions = append(
					entityResolverInputDefinitions,
					entityResolverInputSDL,
				)
			}
		}
	}

	var blocks []string
	if len(entityNames) > 0 {
		entitiesSDL := `# a union of all types that use the @key directive
union _Entity = ` + strings.Join(entityNames, " | ")
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

	// fill in return types for all entity resolvers
	// Entity resolvers always return pointers in Federation
	for _, entity := range f.Entities {
		for _, resolver := range entity.Resolvers {
			// Ensure the return type is a pointer
			if ptr, ok := entity.Type.(*types.Pointer); ok {
				// Already a pointer
				resolver.ReturnType = ptr
			} else {
				// Make it a pointer
				resolver.ReturnType = types.NewPointer(entity.Type)
			}
		}
	}

	explicitRequiresEntities := make(map[string]*Entity, len(requiresEntities))
	for name, e := range requiresEntities {
		if e.IsExplicitRequires() {
			explicitRequiresEntities[name] = e
		}
	}
	if len(explicitRequiresEntities) > 0 {
		err := f.generateExplicitRequires(
			data,
			explicitRequiresEntities,
			requiresImports,
		)
		if err != nil {
			return err
		}
	}

	f.RequiresEntities = requiresEntities

	// Populate ImplDirectives on each entity by extracting the resolved
	// OBJECT-level directives from the corresponding codegen.Object.
	// These are the user-defined directives (e.g. @guard, @auth) that
	// should wrap entity resolver calls — federation-internal directives
	// are excluded.
	for _, e := range f.Entities {
		obj := data.Objects.ByName(e.Def.Name)
		if obj == nil || len(obj.Fields) == 0 {
			continue
		}
		// OBJECT-level directives are propagated to every field during
		// codegen.  Pick them from the first field's directive list.
		for _, d := range obj.Fields[0].Directives {
			if d.SkipRuntime {
				continue
			}
			if d.IsLocation(ast.LocationObject) && !federationDirectiveNames[d.Name] {
				e.ImplDirectives = append(e.ImplDirectives, d)
			}
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
		PruneOptions:    data.Config.GetPruneOptions(),
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

func (f *Federation) buildEntities(schema *ast.Schema, version int) ([]*Entity, error) {
	entities := make([]*Entity, 0)
	for _, schemaType := range schema.Types {
		entity, err := f.buildEntity(schemaType, schema, version)
		if err != nil {
			return nil, err
		}
		if entity != nil {
			entities = append(entities, entity)
		}
	}

	// make sure order remains stable across multiple builds
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].Name < entities[j].Name
	})

	return entities, nil
}

func (f *Federation) buildEntity(
	schemaType *ast.Definition,
	schema *ast.Schema,
	version int,
) (*Entity, error) {
	keys, ok := isFederatedEntity(schemaType)
	if !ok {
		return nil, nil
	}

	if (schemaType.Kind == ast.Interface) && (len(schema.GetPossibleTypes(schemaType)) == 0) {
		fmt.Printf(
			"@key directive found on unused \"interface %s\". Will be ignored.\n",
			schemaType.Name,
		)
		return nil, nil
	}

	multi := f.isMultiEntity(schemaType)
	requiresStrategy, err := f.resolveRequiresStrategy(schemaType, multi)
	if err != nil {
		return nil, err
	}
	if err := validateComputedFields(schemaType, requiresStrategy); err != nil {
		return nil, err
	}

	entity := &Entity{
		Name:             schemaType.Name,
		Def:              schemaType,
		Resolvers:        nil,
		Requires:         nil,
		Multi:            multi,
		RequiresStrategy: requiresStrategy,
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
		return entity, nil
	}

	resolvers, err := buildResolvers(schemaType, schema, keys, entity.Multi)
	if err != nil {
		return nil, err
	}
	entity.Resolvers = resolvers
	entity.Requires = buildRequires(schemaType, entity.RequiresStrategy)
	if len(entity.Requires) > 0 {
		f.usesRequires = true
	}

	return entity, nil
}

// defaultRequiresStrategy is the package-level @requires strategy an entity
// uses when it does not select one via @entityResolver(requires: "…").
func (o PackageOptions) defaultRequiresStrategy() RequiresStrategy {
	switch {
	case o.ComputedRequires:
		return RequiresComputed
	case o.ExplicitRequires:
		return RequiresExplicit
	case o.PreloadedRequires:
		return RequiresPreloaded
	default:
		return RequiresDefault
	}
}

// resolveRequiresStrategy returns the @requires strategy for an entity: the
// @entityResolver(requires: "…") argument if present, otherwise the package
// default. It mirrors isMultiEntity.
//
// Requires: multi is the already-resolved @entityResolver(multi) value.
// Ensures:  the returned strategy is one of the four RequiresStrategy values;
// an unknown directive value, or preloaded on a non-multi entity, is a
// (clear, actionable) error rather than a silent fallback.
func (f *Federation) resolveRequiresStrategy(
	schemaType *ast.Definition,
	multi bool,
) (RequiresStrategy, error) {
	strategy := f.PackageOptions.defaultRequiresStrategy()

	if dir := schemaType.Directives.ForName(dirNameEntityResolver); dir != nil {
		if dirArg := dir.Arguments.ForName("requires"); dirArg != nil {
			dirVal, err := dirArg.Value.Value(nil)
			if err != nil {
				return "", fmt.Errorf(
					"entity %q: reading @entityResolver(requires:): %w",
					schemaType.Name, err,
				)
			}
			parsed, ok := parseRequiresStrategy(dirVal)
			if !ok {
				// computed is deliberately not a directive value: it does not
				// describe how @requires reaches the entity resolver (it routes
				// fields to standalone field resolvers instead), so it does not
				// share the axis the directive selects on. Point users at the
				// package option that does select it rather than a bare
				// "unknown value". See RequiresStrategy in entity.go.
				if dirVal == string(RequiresComputed) {
					return "", fmt.Errorf(
						"entity %q: @entityResolver(requires: %q) is not supported; select the computed strategy with the %q package option instead",
						schemaType.Name,
						RequiresComputed,
						optionComputedRequires,
					)
				}
				return "", fmt.Errorf(
					"entity %q: unknown @entityResolver(requires: %v); valid values are %q, %q, %q",
					schemaType.Name,
					dirVal,
					RequiresDefault,
					RequiresExplicit,
					RequiresPreloaded,
				)
			}
			strategy = parsed
		}
	}

	if strategy == RequiresPreloaded && !multi {
		return "", fmt.Errorf(
			"entity %q: @entityResolver(requires: %q) requires multi: true",
			schemaType.Name, RequiresPreloaded,
		)
	}

	return strategy, nil
}

// parseRequiresStrategy converts a directive argument value to a
// RequiresStrategy, reporting whether it was a recognized value. Only the
// strategies that describe how @requires reaches the entity resolver are
// directive-selectable; RequiresComputed is intentionally excluded (it is
// selected by the computed_requires package option — see RequiresStrategy).
func parseRequiresStrategy(v any) (RequiresStrategy, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	switch RequiresStrategy(s) {
	case RequiresDefault, RequiresExplicit, RequiresPreloaded:
		return RequiresStrategy(s), true
	}
	return "", false
}

// isMultiEntity returns @entityResolver(multi) value, if directive is not defined,
// then global configuration parameter will be used.
func (f *Federation) isMultiEntity(schemaType *ast.Definition) bool {
	dir := schemaType.Directives.ForName(dirNameEntityResolver)
	if dir == nil {
		return f.PackageOptions.EntityResolverMulti
	}

	if dirArg := dir.Arguments.ForName("multi"); dirArg != nil {
		if dirVal, err := dirArg.Value.Value(nil); err == nil {
			return dirVal.(bool)
		}
	}

	return f.PackageOptions.EntityResolverMulti
}

func buildResolvers(
	schemaType *ast.Definition,
	schema *ast.Schema,
	keys []*ast.Directive,
	multi bool,
) ([]*EntityResolver, error) {
	resolvers := make([]*EntityResolver, 0)
	for _, dir := range keys {
		if len(dir.Arguments) > 2 {
			return nil, gqlerror.ErrorPosf(
				dir.Position,
				`@key on %q accepts only the "fields" and "resolvable" arguments`,
				schemaType.Name,
			)
		}
		keyFields, resolverFields, err := buildKeyFields(
			schemaType,
			schema,
			dir,
		)
		if err != nil {
			return nil, err
		}

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

	return resolvers, nil
}

func extractFields(
	dir *ast.Directive,
) (string, error) {
	var arg *ast.Argument

	// since directives are able to now have multiple arguments, we need to check both possible for
	// a possible @key(fields="" fields="")
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
) ([]*KeyField, []string, error) {
	fieldsRaw, err := extractFields(dir)
	if err != nil {
		// api.generate prefixes plugin errors with "federation: ", so the
		// messages here omit that prefix to avoid duplicating it.
		return nil, nil, gqlerror.ErrorPosf(dir.Position, "@key on %q: %s", schemaType.Name, err)
	}

	keyFieldSet := fieldset.New(fieldsRaw, nil)

	keyFields := make([]*KeyField, len(keyFieldSet))
	resolverFields := []string{}
	for i, field := range keyFieldSet {
		def := field.FieldDefinition(schemaType, schema)

		if def == nil {
			return nil, nil, gqlerror.ErrorPosf(
				dir.Position,
				"@key(fields: %q): field %q is not defined on %q",
				fieldsRaw,
				field.Join("."),
				schemaType.Name,
			)
		}

		keyFields[i] = &KeyField{Definition: def, Field: field}
		resolverFields = append(resolverFields, keyFields[i].Field.ToGo())
	}

	assignKeyFieldGoNames(keyFields)

	return keyFields, resolverFields, nil
}

// assignKeyFieldGoNames sets each key field's GoName to a Go identifier that is
// unique within the resolver.
//
// Requires: keyFields are the key fields of a single resolver, in schema order.
// Ensures:  every GoName equals Field.ToGo() when that name is unique among the
// key fields; otherwise the second and later collisions get the smallest
// integer suffix (>= 2) that makes them unique. Assignment is deterministic in
// schema order and mutates only the GoName field. The names are idempotent under
// ToGo, so they are valid as SDL field names, modelgen struct fields, and
// template struct-literal keys alike.
func assignKeyFieldGoNames(keyFields []*KeyField) {
	used := make(map[string]bool, len(keyFields))
	for _, keyField := range keyFields {
		base := keyField.Field.ToGo()
		name := base
		for i := 2; used[name]; i++ {
			name = base + strconv.Itoa(i)
		}
		used[name] = true
		keyField.GoName = name
	}
}

// validateComputedFields rejects unusable @computedRequires placements up front:
// it is only meaningful on a @requires field, and it cannot be combined with the
// explicit strategy, whose Populate<Entity>Requires hook already owns every
// @requires field on the entity.
func validateComputedFields(schemaType *ast.Definition, strategy RequiresStrategy) error {
	for _, field := range schemaType.Fields {
		if field.Directives.ForName(dirNameComputedRequires) == nil {
			continue
		}
		if field.Directives.ForName(dirNameRequires) == nil {
			return fmt.Errorf(
				"entity %q field %q: @computedRequires only applies to @requires fields",
				schemaType.Name, field.Name,
			)
		}
		if strategy == RequiresExplicit {
			return fmt.Errorf(
				"entity %q field %q: @computedRequires cannot be combined with the explicit @requires strategy; the Populate%sRequires hook already handles every @requires field",
				schemaType.Name,
				field.Name,
				schemaType.Name,
			)
		}
	}
	return nil
}

// requiresFieldIsComputed reports whether a field carrying @requires is resolved
// via a standalone field resolver (the computed strategy) rather than through the
// entity resolver: the entity resolves to RequiresComputed (the computed_requires
// package option), or the field is marked @computedRequires. It is the single
// source of truth for the per-field computed decision, read by both buildRequires
// and mutateSchemaForRequires.
func requiresFieldIsComputed(strategy RequiresStrategy, field *ast.FieldDefinition) bool {
	return strategy == RequiresComputed ||
		field.Directives.ForName(dirNameComputedRequires) != nil
}

// buildRequires collects an entity's @requires fields. Each entry's Computed
// flag records whether the field is delivered via a standalone field resolver
// rather than the entity resolver (see requiresFieldIsComputed).
func buildRequires(schemaType *ast.Definition, strategy RequiresStrategy) []*Requires {
	requires := make([]*Requires, 0)
	for _, f := range schemaType.Fields {
		dir := f.Directives.ForName(dirNameRequires)
		if dir == nil {
			continue
		}
		computed := requiresFieldIsComputed(strategy, f)

		fieldsRaw, err := extractFields(dir)
		if err != nil {
			panic("Exactly one `fields` argument needed for @requires declaration.")
		}
		requiresFieldSet := fieldset.New(fieldsRaw, nil)
		for _, field := range requiresFieldSet {
			requires = append(requires, &Requires{
				Name:     field.ToGoPrivate(),
				Field:    field,
				Computed: computed,
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

		populator.Comment = strings.TrimSpace(
			strings.TrimLeft(
				rewriter.GetMethodComment("executionContext", populator.FuncName),
				`\`,
			),
		)
		populator.Implementation = strings.TrimSpace(
			rewriter.GetMethodBody("executionContext", populator.FuncName),
		)

		if populator.Implementation == "" {
			populator.Exists = false
			populator.Implementation = fmt.Sprintf(
				"panic(fmt.Errorf(\"not implemented: %v\"))",
				populator.FuncName,
			)
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
		PruneOptions:    data.Config.GetPruneOptions(),
	})
}

func buildResolverSDL(
	resolver *EntityResolver,
	multi bool,
) (resolverSDL, entityResolverInputSDL string) {
	if multi {
		entityResolverInputSDL = buildEntityResolverInputDefinitionSDL(resolver)
		resolverSDL := fmt.Sprintf(
			"\t%s(reps: [%s]!): [%s]",
			resolver.ResolverName,
			resolver.InputTypeName,
			resolver.ReturnTypeName,
		)
		return resolverSDL, entityResolverInputSDL
	}

	resolverArgs := ""
	var resolverArgsSb705 strings.Builder
	for _, keyField := range resolver.KeyFields {
		fmt.Fprintf(
			&resolverArgsSb705,
			"%s: %s,",
			keyField.Field.ToGoPrivate(),
			keyField.Definition.Type.String(),
		)
	}
	resolverArgs += resolverArgsSb705.String()
	resolverSDL = fmt.Sprintf(
		"\t%s(%s): %s!",
		resolver.ResolverName,
		resolverArgs,
		resolver.ReturnTypeName,
	)
	return resolverSDL, ""
}

func buildEntityResolverInputDefinitionSDL(resolver *EntityResolver) string {
	entityResolverInputDefinition := "input " + resolver.InputTypeName + " {\n"
	var entityResolverInputDefinitionSb714 strings.Builder
	for _, keyField := range resolver.KeyFields {
		fmt.Fprintf(&entityResolverInputDefinitionSb714,
			"\t%s: %s\n",
			keyField.GoName,
			keyField.Definition.Type.String(),
		)
	}
	entityResolverInputDefinition += entityResolverInputDefinitionSb714.String()
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

// mutateSchemaForRequires turns each computed @requires field into a standalone
// field resolver: it forces a resolver for the field and injects the
// _federationRequires argument (carrying the representation via
// @populateFromRepresentations). The decision is per field — a single entity can
// have some computed @requires fields (handled here) and others delivered through
// the entity resolver (handled in the federation template).
//
// It iterates cfg.Schema (not entity.Def): the schema is re-parsed after
// InjectSourcesLate, so entity.Def holds stale field pointers whose mutations
// codegen never sees. The resolved per-entity strategy is looked up by name.
func (f *Federation) mutateSchemaForRequires(cfg *config.Config) {
	strategyByType := make(map[string]RequiresStrategy, len(f.Entities))
	for _, e := range f.Entities {
		strategyByType[e.Def.Name] = e.RequiresStrategy
	}

	for typeName, schemaType := range cfg.Schema.Types {
		strategy, ok := strategyByType[typeName]
		if !ok {
			continue
		}
		for _, field := range schemaType.Fields {
			if field.Directives.ForName(dirNameRequires) == nil {
				continue
			}
			if !requiresFieldIsComputed(strategy, field) {
				continue
			}

			model := cfg.Models[typeName]
			if model.Fields == nil {
				model.Fields = make(map[string]config.TypeMapField)
			}
			fieldConfig := model.Fields[field.Name]
			fieldConfig.Resolver = true
			model.Fields[field.Name] = fieldConfig
			cfg.Models[typeName] = model

			field.Arguments = append(field.Arguments, &ast.ArgumentDefinition{
				Name: fieldArgRequires,
				Type: ast.NamedType(mapTypeName, nil),
				Directives: ast.DirectiveList{
					{
						Name:       dirNamePopulateFromRepresentations,
						Definition: dirPopulateFromRepresentations,
					},
				},
			})
		}
	}
}

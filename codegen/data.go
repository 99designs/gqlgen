package codegen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/codegen/config"
)

// Data is a unified model of the code to be generated. Plugins may modify this structure to do
// things like implement
// resolvers or directives automatically (eg grpc, validation)
type Data struct {
	Config *config.Config
	Schema *ast.Schema
	// If a schema is broken up into multiple Data instance, each representing part of the schema,
	// AllDirectives should contain the directives for the entire schema. Directives() can
	// then be used to get the directives that were defined in this Data instance's sources.
	// If a single Data instance is used for the entire schema, AllDirectives and Directives()
	// will be identical.
	// AllDirectives should rarely be used directly.
	AllDirectives   DirectiveList
	Objects         Objects
	Inputs          Objects
	Interfaces      map[string]*Interface
	ReferencedTypes map[string]*config.TypeReference
	ComplexityRoots map[string]*Object

	QueryRoot        *Object
	MutationRoot     *Object
	SubscriptionRoot *Object
	AugmentedSources []AugmentedSource
	Plugins          []any

	// allBatchFieldTypes is the schema-wide result of collectBatchFieldTypes. Like
	// AllDirectives it is set when a schema is split into one Data instance per schema
	// file, because each instance then holds only the objects declared in its own file
	// and cannot see whether a type declared elsewhere has batch fields.
	allBatchFieldTypes map[string]bool

	// cachedBatchFieldTypes is populated on first use by batchFieldTypes.
	cachedBatchFieldTypes map[string]bool
}

func (d *Data) HasEmbeddableSources() bool {
	hasEmbeddableSources := false
	for _, s := range d.AugmentedSources {
		if s.Embeddable {
			hasEmbeddableSources = true
		}
	}
	return hasEmbeddableSources
}

// collectBatchFieldTypes returns the names of the objects that resolve at least one field
// through a batch resolver.
//
// Root objects are excluded: a batch resolver groups sibling parents, and the root has no
// parent to group — ShortBatchResolverDeclaration returns "" for root fields.
func collectBatchFieldTypes(objects Objects) map[string]bool {
	names := make(map[string]bool)
	for _, obj := range objects {
		if obj.Root {
			continue
		}
		for _, field := range obj.Fields {
			if field.IsBatch() {
				names[obj.Name] = true
				break
			}
		}
	}
	return names
}

// batchFieldTypes returns the set of type names that have batch resolver fields, computing
// it on first use. Plugins may add fields to Data after it is built, so the set is derived
// when a template first asks for it rather than during BuildData.
//
// The cache is not synchronised; it relies on a Data being rendered by one goroutine.
func (d *Data) batchFieldTypes() map[string]bool {
	if d.allBatchFieldTypes != nil {
		return d.allBatchFieldTypes
	}
	if d.cachedBatchFieldTypes == nil {
		d.cachedBatchFieldTypes = collectBatchFieldTypes(d.Objects)
	}
	return d.cachedBatchFieldTypes
}

// HasBatchFields reports whether the named type resolves any field through a batch
// resolver, and so needs batch parents registered when it is marshaled as a list.
func (d *Data) HasBatchFields(typeName string) bool {
	return d.batchFieldTypes()[typeName]
}

// HasBatchResolverFields reports whether any type in the schema uses a batch resolver.
func (d *Data) HasBatchResolverFields() bool {
	return len(d.batchFieldTypes()) > 0
}

// BatchImplementors returns the implementors of the named interface or union whose concrete
// type has batch fields, in declaration order.
//
// Batch parents are grouped by concrete type, so only implementors that resolve a field
// through a batch resolver need collecting. An empty result means the list needs no
// grouping at all.
func (d *Data) BatchImplementors(name string) []InterfaceImplementor {
	iface := d.Interfaces[name]
	if iface == nil {
		return nil
	}
	var impls []InterfaceImplementor
	for _, impl := range iface.Implementors {
		if d.HasBatchFields(impl.Name) {
			impls = append(impls, impl)
		}
	}
	return impls
}

// ChildFieldType represents a unique type referenced as a return type from fields across
// all objects. Each one gets a shared childFields_* function that serves as the
// FieldContext.Child dispatcher of every field returning that type.
type ChildFieldType struct {
	TypeName   string
	Definition *ast.Definition
	// Object is the generated object of an OBJECT type; nil for other kinds.
	Object *Object
}

// ChildFieldTypes collects all unique types that appear as return types from fields,
// sorted by name.
func (d *Data) ChildFieldTypes() []*ChildFieldType {
	objects := make(map[string]*Object, len(d.Objects))
	for _, obj := range d.Objects {
		objects[obj.Name] = obj
	}
	seen := map[string]bool{}
	var result []*ChildFieldType
	for _, obj := range d.Objects {
		for _, field := range obj.Fields {
			if field.TypeReference == nil || field.TypeReference.Definition == nil {
				continue
			}
			def := field.TypeReference.Definition
			if seen[def.Name] {
				continue
			}
			seen[def.Name] = true
			result = append(result, &ChildFieldType{
				TypeName:   def.Name,
				Definition: def,
				Object:     objects[def.Name],
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TypeName < result[j].TypeName
	})
	return result
}

// UniqueChildFieldTypes collects all unique OBJECT types with fields that appear as return
// types from fields, sorted by name.
//
// Deprecated: the templates no longer use it; ChildFieldTypes returns every child type.
func (d *Data) UniqueChildFieldTypes() []*ChildFieldType {
	var result []*ChildFieldType
	for _, ct := range d.ChildFieldTypes() {
		if ct.Definition.Kind == ast.Object && len(ct.Definition.Fields) > 0 {
			result = append(result, ct)
		}
	}
	return result
}

// AugmentedSource contains extra information about graphql schema files which is not known directly
// from the Config.Sources data
type AugmentedSource struct {
	// path relative to Config.Exec.Filename
	RelativePath string
	Embeddable   bool
	BuiltIn      bool
	Source       string
}

type builder struct {
	Config     *config.Config
	Schema     *ast.Schema
	Binder     *config.Binder
	Directives map[string]*Directive
}

// Get only the directives which should have a user provided definition on server instantiation
func (d *Data) UserDirectives() DirectiveList {
	res := DirectiveList{}
	directives := d.Directives()
	for k, directive := range directives {
		if directive.Implementation == nil {
			res[k] = directive
		}
	}
	return res
}

// Get only the directives which should have a statically provided definition
func (d *Data) BuiltInDirectives() DirectiveList {
	res := DirectiveList{}
	directives := d.Directives()
	for k, directive := range directives {
		if directive.Implementation != nil {
			res[k] = directive
		}
	}
	return res
}

// Get only the directives which are defined in the config's sources.
func (d *Data) Directives() DirectiveList {
	res := DirectiveList{}
	for k, directive := range d.AllDirectives {
		for _, s := range d.Config.Sources {
			if directive.Position.Src.Name == s.Name {
				res[k] = directive
				break
			}
		}
	}
	return res
}

// FuncReceiver returns the receiver clause for a generated function declaration.
//
//	function syntax:  ""
//	receiver syntax:  "(ec *executionContext) "
func (d *Data) FuncReceiver() string {
	if d.Config.UseFunctionSyntaxForExecutionContext {
		return ""
	}
	return "(ec *executionContext) "
}

// ECFuncParam returns the ec parameter in a generated function declaration.
//
//	function syntax:  "ec *executionContext, "
//	receiver syntax:  ""
func (d *Data) ECFuncParam() string {
	if d.Config.UseFunctionSyntaxForExecutionContext {
		return "ec *executionContext, "
	}
	return ""
}

// ECDot returns the call-site prefix for a generated function call.
//
//	function syntax:  ""
//	receiver syntax:  "ec."
func (d *Data) ECDot() string {
	if d.Config.UseFunctionSyntaxForExecutionContext {
		return ""
	}
	return "ec."
}

// ECArg returns the ec argument at a generated call site.
//
//	function syntax:  "ec, "
//	receiver syntax:  ""
func (d *Data) ECArg() string {
	if d.Config.UseFunctionSyntaxForExecutionContext {
		return "ec, "
	}
	return ""
}

// ECFuncValue returns the prefix that turns a generated function name into a function value.
// With receiver syntax the value is a method expression, which takes ec as its first
// parameter; ECFuncValueParams and ECFuncValueArgs follow that order.
//
//	function syntax:  ""
//	receiver syntax:  "(*executionContext)."
func (d *Data) ECFuncValue() string {
	if d.Config.UseFunctionSyntaxForExecutionContext {
		return ""
	}
	return "(*executionContext)."
}

// ECFuncValueParams returns the leading parameters of a function type that holds an
// ECFuncValue.
//
//	function syntax:  "ctx context.Context, ec *executionContext, "
//	receiver syntax:  "ec *executionContext, ctx context.Context, "
func (d *Data) ECFuncValueParams() string {
	if d.Config.UseFunctionSyntaxForExecutionContext {
		return "ctx context.Context, ec *executionContext, "
	}
	return "ec *executionContext, ctx context.Context, "
}

// ECFuncValueArgs returns the leading arguments of a call through an ECFuncValue.
//
//	function syntax:  "ctx, ec, "
//	receiver syntax:  "ec, ctx, "
func (d *Data) ECFuncValueArgs() string {
	if d.Config.UseFunctionSyntaxForExecutionContext {
		return "ctx, ec, "
	}
	return "ec, ctx, "
}

// ImplDirectivesField is the subset of Field and FieldArgument needed by the
// implDirectives template.
type ImplDirectivesField interface {
	DirectiveObjName() string
	ImplDirectives() []*Directive
	ZeroVal() string
}

// ImplDirectivesContext is the typed context passed to the implDirectives template.
// ErrVal is the Go variable name to return on error ("zeroVal" for the standard
// case, or the partially-built struct variable for INPUT_OBJECT directives).
// ErrWrap controls whether the error is wrapped in graphql.ErrorOnPath(ctx, err).
type ImplDirectivesContext struct {
	Field   ImplDirectivesField
	Data    *Data
	ErrVal  string
	ErrWrap bool
}

// QueryDirectivesContext is the typed context passed to the queryDirectives template.
type QueryDirectivesContext struct {
	DirectiveList DirectiveList
	Data          *Data
}

// ImplDirectivesCtx returns a context for the implDirectives template suitable
// for field and argument directive processing (ErrVal="zeroVal", ErrWrap=false).
func (d *Data) ImplDirectivesCtx(f ImplDirectivesField) ImplDirectivesContext {
	return ImplDirectivesContext{Field: f, Data: d, ErrVal: "zeroVal", ErrWrap: false}
}

// QueryDirectivesCtx returns a context for the queryDirectives template.
func (d *Data) QueryDirectivesCtx(dl DirectiveList) QueryDirectivesContext {
	return QueryDirectivesContext{DirectiveList: dl, Data: d}
}

// inputObjectImplDirectivesField adapts an *Object for use as an ImplDirectivesField
// in INPUT_OBJECT-level directive processing, where the directive receiver is the
// map representation of the input rather than a typed struct field.
type inputObjectImplDirectivesField struct {
	object *Object
}

func (f *inputObjectImplDirectivesField) DirectiveObjName() string { return "asMap" }
func (f *inputObjectImplDirectivesField) ImplDirectives() []*Directive {
	return f.object.InputObjectDirectives()
}
func (f *inputObjectImplDirectivesField) ZeroVal() string { return "" } // unused when ErrWrap=true

// InputObjectImplDirectivesCtx returns a context for the implDirectives template
// for INPUT_OBJECT-level directive processing. errVal is the Go variable name to
// return on error (e.g., "it" or "&it").
func (d *Data) InputObjectImplDirectivesCtx(obj *Object, errVal string) ImplDirectivesContext {
	return ImplDirectivesContext{
		Field:   &inputObjectImplDirectivesField{object: obj},
		Data:    d,
		ErrVal:  errVal,
		ErrWrap: true,
	}
}

// ErrReturn returns the Go statements for the error path inside a directive
// closure. When ErrWrap is true the error is wrapped in
// graphql.ErrorOnPath; when false the typed zero value is declared before
// the plain error return. errExpr is the Go error expression to return
// (e.g. "err" or `errors.New("not implemented")`).
func (c ImplDirectivesContext) ErrReturn(errExpr string) string {
	if c.ErrWrap {
		return fmt.Sprintf("return %s, graphql.ErrorOnPath(ctx, %s)", c.ErrVal, errExpr)
	}
	return c.Field.ZeroVal() + "\nreturn " + c.ErrVal + ", " + errExpr
}

func BuildData(cfg *config.Config, plugins ...any) (*Data, error) {
	cfg.ReloadAllPackages()

	b := builder{
		Config: cfg,
		Schema: cfg.Schema,
	}

	b.Binder = b.Config.NewBinder()

	var err error
	b.Directives, err = b.buildDirectives()
	if err != nil {
		return nil, err
	}

	dataDirectives := make(map[string]*Directive)
	for name, d := range b.Directives {
		if !d.SkipRuntime {
			dataDirectives[name] = d
		}
	}

	s := Data{
		Config:        cfg,
		AllDirectives: dataDirectives,
		Schema:        b.Schema,
		Interfaces:    map[string]*Interface{},
		Plugins:       plugins,
	}

	for _, schemaType := range b.Schema.Types {
		switch schemaType.Kind {
		case ast.Object:
			obj, err := b.buildObject(schemaType)
			if err != nil {
				return nil, fmt.Errorf("unable to build object definition: %w", err)
			}

			s.Objects = append(s.Objects, obj)
		case ast.InputObject:
			input, err := b.buildObject(schemaType)
			if err != nil {
				return nil, fmt.Errorf("unable to build input definition: %w", err)
			}

			s.Inputs = append(s.Inputs, input)

		case ast.Union, ast.Interface:
			s.Interfaces[schemaType.Name], err = b.buildInterface(schemaType)
			if err != nil {
				return nil, fmt.Errorf("unable to bind to interface: %w", err)
			}
		}
	}

	if s.Schema.Query != nil {
		s.QueryRoot = s.Objects.ByName(s.Schema.Query.Name)
	} else {
		return nil, errors.New("query entry point missing")
	}

	if s.Schema.Mutation != nil {
		s.MutationRoot = s.Objects.ByName(s.Schema.Mutation.Name)
	}

	if s.Schema.Subscription != nil {
		s.SubscriptionRoot = s.Objects.ByName(s.Schema.Subscription.Name)
	}

	if err := b.injectIntrospectionRoots(&s); err != nil {
		return nil, err
	}

	s.ReferencedTypes = b.buildTypes()

	sort.Slice(s.Objects, func(i, j int) bool {
		return s.Objects[i].Name < s.Objects[j].Name
	})
	s.Objects.assignFieldContextIndexes()

	sort.Slice(s.Inputs, func(i, j int) bool {
		return s.Inputs[i].Name < s.Inputs[j].Name
	})

	if b.Binder.SawInvalid {
		// if we have a syntax error, show it
		err := cfg.Packages.Errors()
		if len(err) > 0 {
			return nil, err
		}

		// otherwise show a generic error message
		return nil, errors.New(
			"invalid types were encountered while traversing the go source code, this probably means the invalid code generated isnt correct. add try adding -v to debug",
		)
	}
	var sources []*ast.Source
	sources, err = SerializeTransformedSchema(cfg.Schema, cfg.Sources)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transformed schema: %w", err)
	}

	aSources := []AugmentedSource{}
	for _, s := range sources {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		outputDir := cfg.Exec.Dir()
		sourcePath := filepath.Join(wd, s.Name)
		relative, err := filepath.Rel(outputDir, sourcePath)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to compute path of %s relative to %s: %w",
				sourcePath,
				outputDir,
				err,
			)
		}
		relative = filepath.ToSlash(relative)
		embeddable := true
		if strings.HasPrefix(relative, "..") || s.BuiltIn {
			embeddable = false
		}
		aSources = append(aSources, AugmentedSource{
			RelativePath: relative,
			Embeddable:   embeddable,
			BuiltIn:      s.BuiltIn,
			Source:       s.Input,
		})
	}
	s.AugmentedSources = aSources

	return &s, nil
}

// injectIntrospectionRoots adds __type and __schema to the query object. The query
// type carries gqlparser's definitions of them (see Config.LoadSchema), and
// buildObject skips every field starting with "__", so they are built here from
// those definitions rather than from a copy that could drift from gqlparser's.
func (b *builder) injectIntrospectionRoots(s *Data) error {
	obj := s.Objects.ByName(b.Schema.Query.Name)
	if obj == nil {
		return errors.New("root query type must be defined")
	}

	for _, name := range []string{"__type", "__schema"} {
		def := b.Schema.Query.Fields.ForName(name)
		if def == nil {
			return fmt.Errorf("gqlparser did not add %s to the query type", name)
		}

		field, err := b.buildField(obj, def)
		if err != nil {
			return err
		}
		obj.Fields = append(obj.Fields, field)
	}

	return nil
}

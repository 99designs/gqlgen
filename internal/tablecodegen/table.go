package tablecodegen

import (
	"bytes"
	"errors"
	"fmt"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"github.com/vektah/gqlparser/v2/validator"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
)

// ObjectMarshal returns the Go expression that marshals value, selected by sel, as the
// object, interface or union type called name. value is empty for the root operation
// types. In table mode objects are marshaled by their table; interfaces and unions
// keep their generated function.
func (d *Data) ObjectMarshal(name, sel, value string) string {
	if def := d.Config.Schema.Types[name]; def != nil && def.Kind == ast.Object {
		if value == "" {
			value = "nil"
		}
		return fmt.Sprintf("ec.tables.object%s.Marshal(ctx, ec, %s, %s)", name, sel, value)
	}
	args := "ec, " + sel
	if value != "" {
		args += ", " + value
	}
	return fmt.Sprintf("_%s(ctx, %s)", name, args)
}

// tableLinkedSchema returns schema, the schema that the code is generated for, in SDL,
// when it is not the schema of sources, which the generated code embeds, and "" when it
// is. The schema differs when a SchemaMutator or another plugin changes it after gqlgen
// loads it, and when config.LoadSchema adds an empty query type to a schema without one.
// The functions mode generates code for the changed schema and validates requests against
// the schema of the sources. The runtime of table mode reads from the schema what the
// tables leave to it, such as the directives of the fields, so the generated code links
// the tables to the changed schema then, and still validates requests against that of the
// sources. skipRuntime reports whether the generator handles a directive, which the
// runtime does not run. The SDL is loaded back, so that a schema that it cannot write
// fails the generation rather than the initialization of the generated package.
func tableLinkedSchema(
	schema *ast.Schema,
	sources []*ast.Source,
	skipRuntime func(name string) bool,
) (string, error) {
	parsed, err := gqlparser.LoadSchema(sources...)
	if err != nil {
		return "", fmt.Errorf("failed to load the schema of the sources: %w", err)
	}
	sdl, err := tableSchemaSDL(schema, skipRuntime)
	parsedSDL, parsedErr := tableSchemaSDL(parsed, skipRuntime)
	// The tables are linked to the schema of the sources when it is the schema that the code
	// is generated for: SDL does not write it then, also where it cannot, such as for the
	// directives that the sources put on the types of introspection.
	if sdl == parsedSDL && fmt.Sprint(err) == fmt.Sprint(parsedErr) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if _, err := gqlparser.LoadSchema(
		&ast.Source{Name: "linked_schema.graphql", Input: sdl},
	); err != nil {
		return "", fmt.Errorf(
			"table mode cannot link the tables to the schema that plugins changed: %w",
			err,
		)
	}
	return sdl, nil
}

// tableSources returns sources as the generated code holds them, which augmented
// describes in the same order: those that it embeds from their files under dir, the
// directory of the code, are read from the files, which a program that runs gqlgen may
// have changed in memory.
func tableSources(
	sources []*ast.Source,
	augmented []codegen.AugmentedSource,
	dir string,
) []*ast.Source {
	res := slices.Clone(sources)
	for i, src := range sources {
		if !augmented[i].Embeddable {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(augmented[i].RelativePath)))
		if err != nil {
			// The code does not compile without the file, in either mode.
			continue
		}
		res[i] = &ast.Source{Name: src.Name, Input: string(b), BuiltIn: src.BuiltIn}
	}
	return res
}

// tableEmptyQueryField is the field that tableSchemaSDL writes in the empty query type
// that config.LoadSchema adds to a schema without one, which SDL cannot write without
// fields. The tables have no such field.
const tableEmptyQueryField = "_emptyQueryType"

// tableSchemaSDL writes what Link reads from schema in SDL for tableLinkedSchema, so that
// loading the SDL reads it back and the runtime runs the directives that the functions
// mode runs:
//   - a schema definition names the root operation types, which loading would otherwise
//     take from the types called Query, Mutation and Subscription;
//   - the definitions of the prelude, which loading adds, are left out, but the
//     directives that sources or plugins put on its scalars are written in extensions;
//   - the fields of introspection, which loading adds to the query type, are left out;
//   - the empty query type of config.LoadSchema gets tableEmptyQueryField;
//   - the directives that the runtime never runs are left out: those of the schema and of
//     enum values, those that skipRuntime leaves to the generator, and those that a plugin
//     put where the functions mode does not run them;
//   - a plugin may put a directive where its definition does not allow it but where the
//     functions mode runs it, such as one declared on OBJECT on a field: the definition
//     allows the place in the SDL. A directive that a plugin put more than once on an
//     element, which the functions mode runs each time, may repeat in the SDL.
//
// The functions mode reads where a directive runs from the definition it had when the
// schema was loaded. With the SDL, it returns an error for each directive that the runtime
// would then run elsewhere than the functions mode, and for each that the functions mode
// runs but SDL cannot write: on a field of a type of the prelude, or on a type of
// introspection. The definitions of other built-in sources are kept. The descriptions are
// left out, as the tables do not need them.
func tableSchemaSDL(schema *ast.Schema, skipRuntime func(name string) bool) (string, error) {
	w := &tableSDL{
		schema:      schema,
		skipRuntime: skipRuntime,
		allowed:     map[string][]ast.DirectiveLocation{},
		repeated:    map[string]bool{},
		unions:      map[string][]string{},
	}
	for name, def := range schema.Types {
		if def.Kind == ast.Union {
			for _, member := range def.Types {
				w.unions[member] = append(w.unions[member], name)
			}
		}
	}
	s := ast.Schema{
		Types:      make(map[string]*ast.Definition, len(schema.Types)),
		Directives: make(map[string]*ast.DirectiveDefinition, len(schema.Directives)),
	}
	var extensions ast.DefinitionList
	for _, name := range slices.Sorted(maps.Keys(schema.Types)) {
		def := schema.Types[name]
		switch {
		case fromPrelude(def.Position):
			w.members(def)
			w.preludeFields(def)
			if dirs := w.typeDirectives(def); len(dirs) > 0 {
				if strings.HasPrefix(name, "__") {
					w.fail("%s on %s, a type of introspection", tableDirectives(dirs), name)
				}
				extensions = append(extensions,
					&ast.Definition{Kind: def.Kind, Name: name, Directives: dirs})
			}
		// config.LoadSchema gives the query type that it adds the fields of introspection,
		// which the SDL leaves out.
		case def == schema.Query && !slices.ContainsFunc(def.Fields, func(f *ast.FieldDefinition) bool {
			return !strings.HasPrefix(f.Name, "__")
		}):
			s.Types[name] = &ast.Definition{
				Kind: ast.Object,
				Name: name,
				Fields: ast.FieldList{
					{Name: tableEmptyQueryField, Type: ast.NamedType("Boolean", nil)},
				},
			}
		default:
			w.members(def)
			s.Types[name] = w.definition(def)
		}
	}
	w.check()
	for name, def := range schema.Directives {
		if fromPrelude(def.Position) {
			continue
		}
		if added := w.allowed[name]; len(added) > 0 || w.repeated[name] {
			d := *def
			d.Locations = append(
				slices.Clone(def.Locations),
				slices.Sorted(slices.Values(added))...)
			d.IsRepeatable = d.IsRepeatable || w.repeated[name]
			def = &d
		}
		s.Directives[name] = def
	}

	var buf bytes.Buffer
	// The formatter writes a schema definition only for roots with other names, as s has
	// no roots.
	buf.WriteString("schema {\n")
	for _, root := range []struct {
		op  string
		def *ast.Definition
	}{{"query", schema.Query}, {"mutation", schema.Mutation}, {"subscription", schema.Subscription}} {
		if root.def != nil {
			fmt.Fprintf(&buf, "\t%s: %s\n", root.op, root.def.Name)
		}
	}
	buf.WriteString("}\n")
	f := formatter.NewFormatter(
		&buf,
		formatter.WithNonIntrospectionBuiltin(),
		formatter.WithoutDescription(),
	)
	f.FormatSchema(&s)
	f.FormatSchemaDocument(&ast.SchemaDocument{Extensions: extensions})
	return buf.String(), errors.Join(w.errs...)
}

// tableSDL holds what tableSchemaSDL writes the directives of a schema with.
type tableSDL struct {
	schema      *ast.Schema
	skipRuntime func(name string) bool
	// allowed holds the locations that the SDL adds to the definitions of directives that
	// a plugin put where the definitions do not allow them.
	allowed map[string][]ast.DirectiveLocation
	// repeated holds the directives that a plugin put more than once on an element,
	// whose definitions the SDL lets repeat.
	repeated map[string]bool
	// unions holds the unions that list a type among their members, by the type.
	unions map[string][]string
	// onTypes holds the directives written on types, which check compares with the
	// functions mode.
	onTypes []tableTypeDirective
	// errs holds what SDL cannot write as the functions mode runs it.
	errs []error
}

// tableTypeDirective is a directive on a type, with its definition in the schema and the
// one the functions mode reads where it runs from: see tableLoaded.
type tableTypeDirective struct {
	def, loaded *ast.DirectiveDefinition
	typ         *ast.Definition
}

// tableLoaded returns the definition that the functions mode reads where the directive d
// runs from: the one that d had when the schema was loaded, which a plugin may have
// replaced in the schema since, or def, that of the schema, for a d that a plugin added
// without one.
func tableLoaded(d *ast.Directive, def *ast.DirectiveDefinition) *ast.DirectiveDefinition {
	if d.Definition != nil {
		return d.Definition
	}
	return def
}

// definition returns a copy of def with the directives of def, its fields and their
// arguments that the runtime runs as the functions mode does.
func (w *tableSDL) definition(def *ast.Definition) *ast.Definition {
	d := *def
	d.Directives = w.typeDirectives(def)
	fieldLocation := ast.LocationFieldDefinition
	if def.Kind == ast.InputObject {
		fieldLocation = ast.LocationInputFieldDefinition
	}
	if def.Fields != nil {
		d.Fields = make(ast.FieldList, len(def.Fields))
		for i, field := range def.Fields {
			f := *field
			// The functions mode runs the directives on a field that are declared on the
			// field's location, on objects or on input objects (Field.ImplDirectives).
			f.Directives = w.own(field.Directives, fieldLocation,
				fieldLocation, ast.LocationObject, ast.LocationInputObject)
			if field.Arguments != nil {
				f.Arguments = make(ast.ArgumentDefinitionList, len(field.Arguments))
				for j, arg := range field.Arguments {
					a := *arg
					a.Directives = w.own(arg.Directives, ast.LocationArgumentDefinition,
						ast.LocationArgumentDefinition)
					f.Arguments[j] = &a
				}
			}
			d.Fields[i] = &f
		}
	}
	if def.EnumValues != nil {
		d.EnumValues = make(ast.EnumValueList, len(def.EnumValues))
		for i, value := range def.EnumValues {
			v := *value
			v.Directives = nil
			d.EnumValues[i] = &v
		}
	}
	return &d
}

// own returns the directives of dirs, on an element at location, that the functions mode
// runs there: those whose definitions declare one of runs. The definitions of those allow
// location in the SDL.
func (w *tableSDL) own(
	dirs ast.DirectiveList,
	location ast.DirectiveLocation,
	runs ...ast.DirectiveLocation,
) ast.DirectiveList {
	var res ast.DirectiveList
	for _, d := range dirs {
		def := w.schema.Directives[d.Name]
		if def == nil || w.skipRuntime(d.Name) ||
			!w.declares(tableLoaded(d, def), false, runs...) {
			continue
		}
		w.allow(def, location)
		w.repeat(def, res)
		res = append(res, d)
	}
	return res
}

// typeDirectives returns the directives of def, a type, that the functions mode runs
// somewhere. The definitions of those allow the location of the type in the SDL.
func (w *tableSDL) typeDirectives(def *ast.Definition) ast.DirectiveList {
	var res ast.DirectiveList
	for _, d := range def.Directives {
		dd := w.schema.Directives[d.Name]
		if dd == nil || w.skipRuntime(d.Name) {
			continue
		}
		loaded := tableLoaded(d, dd)
		if tableTypeRunsOf(def.Kind, w.declarer(loaded, false), true) == (tableTypeRuns{}) {
			continue
		}
		w.allow(dd, tableTypeLocation(def.Kind))
		w.repeat(dd, res)
		w.onTypes = append(w.onTypes, tableTypeDirective{def: dd, loaded: loaded, typ: def})
		res = append(res, d)
	}
	return res
}

// members fails for an object, def, whose interfaces and unions the map of the schema lists
// otherwise than the definitions: the functions mode reads them from the map, to collect
// the fragments on them, and the runtime from the schema it links the tables to, which
// loads them from the definitions that SDL writes, or from the sources. Only a plugin that
// changes one and not the other makes them differ.
func (w *tableSDL) members(def *ast.Definition) {
	if def.Kind != ast.Object {
		return
	}
	var functions []string
	for _, i := range w.schema.GetImplements(def) {
		functions = append(functions, i.Name)
	}
	runtime := slices.Concat(def.Interfaces, w.unions[def.Name])
	slices.Sort(functions)
	slices.Sort(runtime)
	if !slices.Equal(slices.Compact(functions), slices.Compact(runtime)) {
		w.fail(
			"the interfaces and unions of %s, which the schema lists otherwise than the definitions,",
			def.Name,
		)
	}
}

// preludeFields fails for the fields that the sources or a plugin added to def, a type of
// the prelude, and for the directives on its fields and arguments where the functions mode
// runs them: SDL cannot write them. The errors write the directives with their arguments,
// as tableLinkedSchema compares them.
func (w *tableSDL) preludeFields(def *ast.Definition) {
	if len(def.Interfaces) > 0 {
		w.fail(
			"%s implements %s, added to a built-in type,",
			def.Name,
			strings.Join(def.Interfaces, " & "),
		)
	}
	for _, f := range def.Fields {
		if !fromPrelude(f.Position) {
			var args []string
			for _, a := range f.Arguments {
				args = append(args, a.Name+": "+a.Type.String())
			}
			w.fail("the field %s.%s(%s): %s, added to a built-in type,",
				def.Name, f.Name, strings.Join(args, ", "), f.Type.String())
		}
		dirs := w.own(f.Directives, ast.LocationFieldDefinition,
			ast.LocationFieldDefinition, ast.LocationObject, ast.LocationInputObject)
		if len(dirs) > 0 {
			w.fail(
				"%s on %s.%s, a field of a built-in type",
				tableDirectives(dirs),
				def.Name,
				f.Name,
			)
		}
		for _, a := range f.Arguments {
			dirs := w.own(
				a.Directives,
				ast.LocationArgumentDefinition,
				ast.LocationArgumentDefinition,
			)
			if len(dirs) > 0 {
				w.fail("%s on %s.%s(%s:), an argument of a built-in type",
					tableDirectives(dirs), def.Name, f.Name, a.Name)
			}
		}
	}
}

// check fails for a directive on a type that the runtime would run elsewhere than the
// functions mode, with the locations that the SDL adds to its definition.
func (w *tableSDL) check() {
	for _, t := range w.onTypes {
		functions := tableTypeRunsOf(t.typ.Kind, w.declarer(t.loaded, false), true)
		runtime := tableTypeRunsOf(t.typ.Kind, w.declarer(t.def, true), false)
		switch {
		case functions == runtime:
		case t.loaded != t.def:
			w.fail("@%s on %s, whose definition a plugin replaced", t.def.Name, t.typ.Name)
		default:
			w.fail("@%s on %s, where its definition does not allow it", t.def.Name, t.typ.Name)
		}
	}
}

// repeat lets the definition of def repeat in the SDL when dirs, the directives before it
// on an element, already hold it, which the functions mode runs again.
func (w *tableSDL) repeat(def *ast.DirectiveDefinition, dirs ast.DirectiveList) {
	if def.IsRepeatable || dirs.ForName(def.Name) == nil {
		return
	}
	if fromPrelude(def.Position) {
		w.fail("@%s more than once on an element, which its definition does not allow", def.Name)
		return
	}
	w.repeated[def.Name] = true
}

// allow lets the definition of def allow location in the SDL.
func (w *tableSDL) allow(def *ast.DirectiveDefinition, location ast.DirectiveLocation) {
	if w.declares(def, true, location) {
		return
	}
	if fromPrelude(def.Position) {
		// SDL cannot declare a directive of the prelude again.
		w.fail("@%s on %s, where its definition does not allow it", def.Name, location)
		return
	}
	w.allowed[def.Name] = append(w.allowed[def.Name], location)
}

// declares reports whether def declares one of locations, with those that the SDL adds
// when added is set.
func (w *tableSDL) declares(
	def *ast.DirectiveDefinition,
	added bool,
	locations ...ast.DirectiveLocation,
) bool {
	for _, l := range locations {
		if slices.Contains(def.Locations, l) || added && slices.Contains(w.allowed[def.Name], l) {
			return true
		}
	}
	return false
}

func (w *tableSDL) declarer(
	def *ast.DirectiveDefinition,
	added bool,
) func(...ast.DirectiveLocation) bool {
	return func(locations ...ast.DirectiveLocation) bool { return w.declares(def, added, locations...) }
}

func (w *tableSDL) fail(format string, args ...any) {
	w.errs = append(w.errs, fmt.Errorf(
		"table mode cannot run "+format+" as the functions mode does; generate without -tags gqlgen_table",
		args...))
}

// tableDirectives writes dirs with their arguments.
func tableDirectives(dirs ast.DirectiveList) string {
	var b strings.Builder
	for i, d := range dirs {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString("@" + d.Name)
		for j, a := range d.Arguments {
			sep := ", "
			if j == 0 {
				sep = "("
			}
			fmt.Fprintf(&b, "%s%s: %s", sep, a.Name, a.Value.String())
		}
		if len(d.Arguments) > 0 {
			b.WriteString(")")
		}
	}
	return b.String()
}

// tableTypeRuns tells where a directive on a type runs: around the fields that return the
// type, around the input fields of the type, and on the input object itself.
type tableTypeRuns struct{ fields, inputFields, input bool }

// tableTypeRunsOf returns where the functions mode, or the runtime when functions is unset,
// runs a directive on a type of kind, whose definition declares what declares reports.
// Both run the directives of the type around a field that are not declared on input
// objects and are declared on the field's location or on objects; the functions mode runs
// those on an input object that are declared on input objects, and the runtime every one
// the SDL writes.
func tableTypeRunsOf(
	kind ast.DefinitionKind,
	declares func(...ast.DirectiveLocation) bool,
	functions bool,
) tableTypeRuns {
	notInput := !declares(ast.LocationInputObject)
	var r tableTypeRuns
	if kind != ast.InputObject {
		r.fields = notInput && declares(ast.LocationFieldDefinition, ast.LocationObject)
	}
	if kind == ast.Scalar || kind == ast.Enum || kind == ast.InputObject {
		r.inputFields = notInput && declares(ast.LocationInputFieldDefinition, ast.LocationObject)
	}
	if kind == ast.InputObject {
		r.input = !functions || declares(ast.LocationInputObject)
	}
	return r
}

// tableTypeLocation returns the location of the directives of a type of kind.
func tableTypeLocation(kind ast.DefinitionKind) ast.DirectiveLocation {
	switch kind {
	case ast.Scalar:
		return ast.LocationScalar
	case ast.Object:
		return ast.LocationObject
	case ast.Interface:
		return ast.LocationInterface
	case ast.Union:
		return ast.LocationUnion
	case ast.Enum:
		return ast.LocationEnum
	case ast.InputObject:
		return ast.LocationInputObject
	}
	return ""
}

// fromPrelude reports whether pos is in the prelude of gqlparser, which loading a schema
// adds: the built-in source called as the prelude, of which the schema may hold a copy.
func fromPrelude(pos *ast.Position) bool {
	return pos != nil && pos.Src != nil && pos.Src.BuiltIn && pos.Src.Name == validator.Prelude.Name
}

// tableKind is how table mode builds the marshaler or unmarshaler of a type reference.
type tableKind int

const (
	// tableKindFunc keeps the function the functions mode generates. The tables declare
	// its tableimpl.Out or tableimpl.In when a field returns the type or an argument takes it.
	tableKindFunc tableKind = iota
	// tableKindTyped builds it from the runtime's combinators, which are generic in the Go
	// type, and keeps it with its tableimpl.Out or tableimpl.In in an tableimpl.OutFor or tableimpl.InFor.
	tableKindTyped
	// tableKindErased builds its tableimpl.Out or tableimpl.In with a constructor of the runtime
	// that is not generic in the Go type, from a nil pointer to the type, or in one
	// instantiation for a list. The compiler instantiates a generic function once per
	// type argument, and a schema has a Go type per object, enum and input, so the
	// objects marshaled through a pointer, the types that marshal themselves, the inputs,
	// the named strings bound to a String scalar and the lists are erased, unless the
	// generated code calls their typed marshaler or unmarshaler. See tableTypedCalls. The
	// runtime derives most of them itself, from the Go type: see tableDerived.
	tableKindErased
)

// tableMarshalKind returns how table mode builds the marshaler of t.
func (d *Data) tableMarshalKind(t *config.TypeReference) tableKind {
	kind := d.tableMarshalBase(t)
	if kind == tableKindErased && d.tableInfo().typedCalls[t.MarshalFunc()] {
		if !tableTypedCombinator(t) {
			return tableKindFunc
		}
		return tableKindTyped
	}
	return kind
}

// tableTypedCombinator reports whether the runtime has a typed combinator for t, which
// the generated code calls where it needs the typed marshaler or unmarshaler of an
// erased type. A list always has one. A cast has none, nor has a named slice or map
// type that marshals itself, whose nil the typed combinators do not check: such a type
// keeps its generated function then. A list whose elements marshal themselves carries
// the IsMarshaler of its elements, which does not count.
func tableTypedCombinator(t *config.TypeReference) bool {
	return t.IsSlice() || (!tableCast(t) && (!t.IsMarshaler || !t.IsNilable() || t.IsPtr()))
}

// tableCast reports whether t is a named type whose underlying type is string, bound to a
// String scalar or an enum, which the functions mode marshals with the functions of
// string in a function generated per type that converts the value. The binder sets
// CastType for such types only, but a type bound to a target that marshals itself keeps
// the CastType of its model and loses the functions.
func tableCast(t *config.TypeReference) bool {
	basic, ok := t.CastType.(*types.Basic)
	return ok && basic.Kind() == types.String && !t.IsMarshaler &&
		t.Marshaler != nil && t.Unmarshaler != nil
}

// tableFuncFits reports whether fn, the function that t is bound to, has the type that the
// combinators of the runtime take: func(V) graphql.Marshaler, or graphql.ContextMarshaler
// for a context marshaler, to marshal, and func(any) (V, error), with a context.Context
// first for a context unmarshaler, to unmarshal, where V is the Go type of t, or the type
// it points to. The functions mode calls fn and returns its result, which compiles for
// other types the values can be assigned to: such a function keeps the generated one. The
// types are compared by their names, as fn may come from another load of the packages
// than the Go type of t.
func tableFuncFits(t *config.TypeReference, fn *types.Func, marshal bool) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok && fn.Pkg() != nil {
		// The binder names the unmarshaler of a scalar bound to functions without its type.
		if obj, isFunc := fn.Pkg().Scope().Lookup(fn.Name()).(*types.Func); isFunc {
			sig, ok = obj.Type().(*types.Signature)
		}
	}
	if !ok || sig.Variadic() || sig.Recv() != nil {
		return false
	}
	v := t.GO
	if ptr, ok := v.(*types.Pointer); ok {
		v = ptr.Elem()
	}
	params, results := sig.Params(), sig.Results()
	if marshal {
		want := "Marshaler"
		if t.IsContext {
			want = "ContextMarshaler"
		}
		return params.Len() == 1 && results.Len() == 1 &&
			tableSameType(params.At(0).Type(), v) &&
			tableNamed(results.At(0).Type(), "github.com/99designs/gqlgen/graphql", want)
	}
	n := 1
	if t.IsContext {
		n = 2
		if params.Len() != n || !tableNamed(params.At(0).Type(), "context", "Context") {
			return false
		}
	}
	return params.Len() == n && results.Len() == 2 &&
		tableEmptyInterface(params.At(n-1).Type()) &&
		tableSameType(results.At(0).Type(), v) &&
		tableSameType(results.At(1).Type(), types.Universe.Lookup("error").Type())
}

// tableEmptyInterface reports whether t is the empty interface without a name, such as any.
func tableEmptyInterface(t types.Type) bool {
	iface, ok := types.Unalias(t).(*types.Interface)
	return ok && iface.Empty()
}

// tableSameType reports whether a and b are the same type by their names.
func tableSameType(a, b types.Type) bool {
	return types.TypeString(types.Unalias(a), nil) == types.TypeString(types.Unalias(b), nil)
}

// tableNamed reports whether t is the named type name of the package at path.
func tableNamed(t types.Type, path, name string) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == path &&
		named.Obj().Name() == name
}

// tableOwnType reports whether t, a reference to an object or an input, is bound to the Go
// type of the object or input. A reference may be bound to another model of the type, such
// as the second one of @goModel, which the functions mode does not compile: such a
// reference keeps the function that the functions mode generates, which checks the Go type
// in the same way. See TableGoType.
func (d *Data) tableOwnType(t *config.TypeReference) bool {
	goType := d.tableInfo().goTypes[t.Definition.Name]
	return goType != nil && t.Target != nil && types.Identical(t.Target, goType)
}

// TableGoType returns the Go type of the object or input called name, which the functions
// mode marshals and unmarshals: the first of its models.
func (d *Data) TableGoType(name string) types.Type {
	return d.tableInfo().goTypes[name]
}

// TableObjectReference returns the Go type in which the functions mode passes the object
// called name to its marshaler: a pointer to it, unless it can be nil.
func (d *Data) TableObjectReference(name string) types.Type {
	t := d.TableGoType(name)
	if config.IsNilable(t) {
		return t
	}
	return types.NewPointer(t)
}

// TableMarshalReference returns the statements that the function that table mode keeps for
// t, a reference to an object, interface or union, returns the marshaled value v with. See
// tableMarshalObject.
func (d *Data) TableMarshalReference(t *config.TypeReference, v string) string {
	value := t.GO
	if !t.IsNilable() {
		value = types.NewPointer(t.GO)
	}
	return d.tableMarshalObject(t.Definition.Name, value, v, "obj")
}

// TableMarshalImplementor is TableMarshalReference for the case of the implementation i in
// the marshaler of an interface or a union, which marshals obj.
func (d *Data) TableMarshalImplementor(i codegen.InterfaceImplementor, obj string) string {
	value := i.Type
	if i.TakeRef {
		value = types.NewPointer(i.Type)
	}
	return d.tableMarshalObject(i.Name, value, obj, "o")
}

// tableMarshalObject returns the statements that return v, a value of Go type value, marshaled
// as the type called name. An object is marshaled by its table, which takes it as any. Where
// the value does not have the Go type in which the functions mode passes it to the marshaler
// of the object, it is first assigned to a variable called local of that type: the
// assignment converts a value that can be assigned to that type, as the functions mode does,
// which the resolve function of the object asserts, and fails to compile for the second
// model of @goModel, as the functions mode does. Interfaces and unions keep their typed
// function.
func (d *Data) tableMarshalObject(name string, value types.Type, v, local string) string {
	if !d.tableChecksObject(name, value) {
		return "return " + d.ObjectMarshal(name, "sel", v)
	}
	return fmt.Sprintf("var %s %s = %s\nreturn %s",
		local, tableType(d.TableObjectReference(name)), v,
		d.ObjectMarshal(name, "sel", local))
}

// tableChecksObject reports whether tableMarshalObject assigns a value of Go type value to
// the Go type in which the functions mode passes the type called name to its marshaler: only
// for an object, where the value does not have that type.
func (d *Data) tableChecksObject(name string, value types.Type) bool {
	def := d.Config.Schema.Types[name]
	return def != nil && def.Kind == ast.Object &&
		!types.Identical(value, d.TableObjectReference(name))
}

// tableMarshalBase is tableMarshalKind before the calls of the generated code are taken
// into account.
func (d *Data) tableMarshalBase(t *config.TypeReference) tableKind {
	if !d.tableCombinable(t) || t.IsRoot || (t.IsContext && t.Marshaler == nil) ||
		d.tableBatchList(t) {
		return tableKindFunc
	}
	ptr, isPtr := t.GO.(*types.Pointer)
	switch {
	case t.IsSlice():
		return tableKindErased
	case t.IsMarshaler:
		if !types.IsInterface(t.GO) {
			return tableKindErased
		}
	case tableCast(t):
		if !t.IsContext && (isPtr || !t.IsNilable()) {
			return tableKindErased
		}
	case t.Marshaler != nil:
		if !t.IsTargetNilable() && (isPtr || !t.IsNilable()) &&
			tableFuncFits(t, t.Marshaler, true) {
			return tableKindTyped
		}
	case t.Definition.Kind == ast.Object:
		if t.Target == nil || config.IsNilable(t.Target) || !d.tableOwnType(t) {
			return tableKindFunc
		}
		if types.Identical(t.GO, t.Target) {
			return tableKindTyped
		}
		if isPtr && types.Identical(ptr.Elem(), t.Target) {
			return tableKindErased
		}
	case t.Definition.Kind == ast.Interface || t.Definition.Kind == ast.Union:
		// A reference bound to another model, such as an interface that embeds that of
		// the type, keeps the function, which takes it as the functions mode does.
		if types.IsInterface(t.GO) &&
			types.Identical(t.GO, d.tableInfo().goTypes[t.Definition.Name]) {
			return tableKindTyped
		}
	}
	return tableKindFunc
}

// tableBatchList reports whether the marshaler of the list t registers its elements as
// the parents of batch resolvers, which the function of the functions mode does: see
// type.gotpl. Table mode keeps the function, so that the batch resolvers receive the same
// parents.
func (d *Data) tableBatchList(t *config.TypeReference) bool {
	return t.IsSlice() && !t.Elem().IsSlice() && d.tableInfo().batchParents[t.Definition.Name]
}

// TableMarshal returns the Go expression that builds the marshaler of t for the tables in
// table mode: an tableimpl.OutFor from the runtime's combinators, or an tableimpl.Out from a
// constructor that is not generic in the Go type. It returns "" when t keeps the function
// the functions mode generates.
func (d *Data) TableMarshal(t *config.TypeReference) string {
	switch d.tableMarshalKind(t) {
	case tableKindTyped:
		return "tableimpl.OutOf(" + d.tableMarshalTyped(t) + ")"
	case tableKindErased:
		return d.tableMarshalErased(t)
	}
	return ""
}

// GeneratesMarshaler reports whether the generated code declares the function of the
// marshaler of t: the functions mode always does, and table mode where it keeps the
// function that the functions mode generates.
func (d *Data) GeneratesMarshaler(t *config.TypeReference) bool {
	return d.tableMarshalKind(t) == tableKindFunc
}

// TableMarshalChecks returns the statements that check, when the package compiles, that
// the Go types of d that marshal themselves are graphql.Marshalers, as the functions mode
// returns them, for those that table mode does not keep the marshalers of: a method of a
// pointer receiver on a value, or of another signature, fails to compile as in the
// functions mode. The template puts them into a function without parameters that is not
// called: the marshalers of the functions mode name the type only in their signatures.
func (d *Data) TableMarshalChecks() []string {
	var checks []string
	for _, t := range d.tableSelfMarshalers() {
		if t.MarshalFunc() == "" || d.GeneratesMarshaler(t) {
			continue
		}
		iface := "graphql.Marshaler"
		if t.IsContext {
			iface = "graphql.ContextMarshaler"
		}
		checks = append(checks, "var _ "+iface+" = *new("+
			templates.CurrentImports.LookupType(t.GO)+")")
	}
	slices.Sort(checks)
	return slices.Compact(checks)
}

// TableUnmarshalChecks is TableMarshalChecks for the unmarshalers, which call UnmarshalGQL
// with an any. The template puts them into a function with the parameters ec, ctx and v,
// as the unmarshalers of the functions mode have, which hide packages of those names in
// both.
func (d *Data) TableUnmarshalChecks() []string {
	var checks []string
	for _, t := range d.tableSelfMarshalers() {
		if t.UnmarshalFunc() == "" || t.Unmarshaler != nil || d.GeneratesUnmarshaler(t) {
			continue
		}
		// The functions mode calls the method on a variable of the type, or on a new
		// value of the type it points to, of which a pointer has the same methods.
		res := t.GO
		if t.IsNilable() && t.Elem() != nil {
			res = t.Elem().GO
		}
		call := ".UnmarshalGQL(v)"
		if t.IsContext {
			call = ".UnmarshalGQLContext(ctx, v)"
		}
		checks = append(checks, "_ = graphql.ErrorOnPath(ctx, new("+
			templates.CurrentImports.LookupType(res)+")"+call+")")
	}
	// The references of a type and of a pointer to it check the same unmarshaler.
	slices.Sort(checks)
	return slices.Compact(checks)
}

// tableSelfMarshalers returns the type references of d that marshal or unmarshal
// themselves, other than the lists and pointers to them, whose marshalers and
// unmarshalers call those of their elements, in the order of their keys.
func (d *Data) tableSelfMarshalers() []*config.TypeReference {
	var refs []*config.TypeReference
	for _, key := range slices.Sorted(maps.Keys(d.ReferencedTypes)) {
		t := d.ReferencedTypes[key]
		if t.IsMarshaler && !t.IsSlice() && !t.IsPtrToSlice() && !t.IsPtrToIntf() {
			refs = append(refs, t)
		}
	}
	return refs
}

// GeneratesUnmarshaler is GeneratesMarshaler for the unmarshaler of t.
func (d *Data) GeneratesUnmarshaler(t *config.TypeReference) bool {
	return d.tableUnmarshalKind(t) == tableKindFunc
}

// tableListOptions returns the tableimpl.ListOptions of the list type reference t.
func (d *Data) tableListOptions(t *config.TypeReference) string {
	var opts []string
	if !t.GQL.NonNull {
		opts = append(opts, "Nullable: true")
	}
	if t.Elem().GQL.NonNull {
		opts = append(opts, "ElemNonNull: true")
	}
	if t.IsScalar() {
		opts = append(opts, "Leaf: true")
	}
	if d.Config.Exec.WorkerLimit != 0 {
		opts = append(opts, fmt.Sprintf("WorkerLimit: %d", d.Config.Exec.WorkerLimit))
	}
	if d.Config.OmitPanicHandler {
		opts = append(opts, "OmitPanicHandler: true")
	}
	return "tableimpl.ListOptions{" + strings.Join(opts, ", ") + "}"
}

// tableMarshalTyped returns the Go expression of the runtime combinator that marshals t,
// of type tableimpl.Marshal, for the kinds of t that have one.
func (d *Data) tableMarshalTyped(t *config.TypeReference) string {
	typ := strconv.Quote(t.GQL.String())
	ptr, isPtr := t.GO.(*types.Pointer)
	switch {
	case t.IsSlice():
		elem := t.Elem()
		if d.tableMarshalKind(elem) == tableKindErased && elem.IsNilable() {
			// The elements are marshaled by their Out: a pointer costs nothing to pass as
			// any, and the Out of the element is not generic in its type.
			return fmt.Sprintf(
				"tableimpl.MarshalListOf[%s](%s, %s)",
				tableType(elem.GO), d.TableOut(elem), d.tableListOptions(t),
			)
		}
		return fmt.Sprintf(
			"tableimpl.MarshalList(%s, %s)",
			d.tableElemMarshal(elem),
			d.tableListOptions(t),
		)
	case t.IsMarshaler:
		if isPtr {
			return fmt.Sprintf(
				"tableimpl.MarshalSelfPtr[*executionContext, %s](%s)",
				tableType(ptr.Elem()),
				typ,
			)
		}
		return fmt.Sprintf("tableimpl.MarshalSelf[*executionContext, %s]()", tableType(t.GO))
	case t.Marshaler != nil:
		fn := "tableimpl.MarshalFunc"
		if t.IsContext {
			fn = "tableimpl.MarshalFuncContext"
		}
		if isPtr {
			fn += "Ptr"
		}
		return fmt.Sprintf("%s[*executionContext](%s, %s)", fn, typ, templates.Call(t.Marshaler))
	case t.Definition.Kind == ast.Object:
		if isPtr {
			return fmt.Sprintf(
				"tableimpl.MarshalObjectPtr[*executionContext, %s](%s, &t.object%s)",
				tableType(t.GO), typ, t.Definition.Name,
			)
		}
		return fmt.Sprintf(
			"tableimpl.MarshalObject[*executionContext, %s](&t.object%s)",
			tableType(t.GO),
			t.Definition.Name,
		)
	case t.Definition.Kind == ast.Interface || t.Definition.Kind == ast.Union:
		return fmt.Sprintf("tableimpl.MarshalInterface(%s, _%s)", typ, t.Definition.Name)
	}
	panic("codegen: no typed marshaler for " + t.MarshalFunc())
}

// tableMarshalErased returns the Go expression that builds the tableimpl.Out of t without a
// type parameter for its Go type, for the kinds of t that have one. The type is given by
// a nil pointer to it.
func (d *Data) tableMarshalErased(t *config.TypeReference) string {
	typ := strconv.Quote(t.GQL.String())
	switch {
	case t.IsSlice():
		elem := t.Elem()
		if d.tableMarshalKind(elem) == tableKindErased && elem.IsNilable() {
			// The elements are marshaled by their Out: a pointer costs nothing to pass as
			// any, and the Out of the element is not generic in its type.
			return fmt.Sprintf(
				"tableimpl.ListOutOf[%s](%s, %s)",
				tableType(elem.GO), d.TableOut(elem), d.tableListOptions(t),
			)
		}
		return fmt.Sprintf(
			"tableimpl.ListOut(%s, %s)",
			d.tableElemMarshal(elem),
			d.tableListOptions(t),
		)
	case tableCast(t):
		if t.IsPtr() {
			return fmt.Sprintf(
				"tableimpl.CastPtrOut[*executionContext](%s, %s, (%s)(nil))",
				typ, templates.Call(t.Marshaler), tableType(t.GO),
			)
		}
		return fmt.Sprintf(
			"tableimpl.CastOut[*executionContext](%s, %s, (*%s)(nil))",
			typ, templates.Call(t.Marshaler), tableType(t.GO),
		)
	case t.IsMarshaler:
		if t.IsPtr() {
			return fmt.Sprintf(
				"tableimpl.SelfPtrOut[*executionContext](%s, (%s)(nil))",
				typ,
				tableType(t.GO),
			)
		}
		return fmt.Sprintf(
			"tableimpl.SelfOut[*executionContext](%s, (*%s)(nil))",
			typ,
			tableType(t.GO),
		)
	case t.Definition.Kind == ast.Object:
		return fmt.Sprintf(
			"tableimpl.ObjectPtrOut(%s, &t.object%s, (%s)(nil))",
			typ, t.Definition.Name, tableType(t.GO),
		)
	}
	panic("codegen: no erased marshaler for " + t.MarshalFunc())
}

// tableElemMarshal returns the typed marshaler of elem, the element of a list, for
// tableimpl.MarshalList: the Marshal of its tableimpl.OutFor, the generated function, or for an
// erased type the combinator, instantiated for the list alone.
func (d *Data) tableElemMarshal(elem *config.TypeReference) string {
	switch d.tableMarshalKind(elem) {
	case tableKindTyped:
		return "t." + d.tableName(elem.MarshalFunc()) + ".Marshal"
	case tableKindErased:
		return d.tableMarshalTyped(elem)
	}
	return elem.MarshalFunc()
}

// tableUnmarshalKind returns how table mode builds the unmarshaler of t.
func (d *Data) tableUnmarshalKind(t *config.TypeReference) tableKind {
	kind := d.tableUnmarshalBase(t)
	if kind == tableKindErased && d.tableInfo().typedCalls[t.UnmarshalFunc()] {
		if !tableTypedCombinator(t) {
			return tableKindFunc
		}
		return tableKindTyped
	}
	return kind
}

// tableUnmarshalBase is tableUnmarshalKind before the calls of the generated code are
// taken into account.
func (d *Data) tableUnmarshalBase(t *config.TypeReference) tableKind {
	if !d.tableCombinable(t) || (t.IsContext && t.Unmarshaler == nil) {
		return tableKindFunc
	}
	isPtr := t.IsPtr()
	switch {
	case t.IsSlice():
		return tableKindErased
	case tableCast(t):
		if !t.IsContext && (isPtr || !t.IsNilable()) {
			return tableKindErased
		}
	case t.Unmarshaler != nil:
		if !t.IsTargetNilable() && (isPtr || !t.IsNilable()) &&
			tableFuncFits(t, t.Unmarshaler, false) {
			return tableKindTyped
		}
	case t.IsMarshaler:
		if !types.IsInterface(t.GO) {
			return tableKindErased
		}
	case t.Definition.Kind == ast.InputObject:
		// An input that unmarshals itself without marshaling itself has no table, and
		// the function refers to it, which fails to compile as in the functions mode.
		if t.PointersInUnmarshalInput || t.IsMap() || !d.tableOwnType(t) ||
			d.tableInfo().unmarshalsItself[t.Definition.Name] {
			return tableKindFunc
		}
		if isPtr || !t.IsNilable() {
			return tableKindErased
		}
	}
	return tableKindFunc
}

// TableUnmarshal is TableMarshal for the unmarshaler of t.
func (d *Data) TableUnmarshal(t *config.TypeReference) string {
	switch d.tableUnmarshalKind(t) {
	case tableKindTyped:
		return "tableimpl.InOf(" + d.tableUnmarshalTyped(t) + ")"
	case tableKindErased:
		return d.tableUnmarshalErased(t)
	}
	return ""
}

// tableNullable reports whether the functions mode returns nil for null before anything
// else in the unmarshaler of t.
func tableNullable(t *config.TypeReference) bool {
	return t.IsNilable() && !t.GQL.NonNull
}

// tableUnmarshalTyped returns the Go expression of the runtime combinator that unmarshals
// t, of type tableimpl.Unmarshal, for the kinds of t that have one.
func (d *Data) tableUnmarshalTyped(t *config.TypeReference) string {
	nullable := tableNullable(t)
	ptr, isPtr := t.GO.(*types.Pointer)
	switch {
	case t.IsSlice():
		elem := t.Elem()
		if d.tableUnmarshalKind(elem) == tableKindErased && elem.IsNilable() {
			return fmt.Sprintf(
				"tableimpl.UnmarshalListOf[%s](%t, %s)",
				tableType(elem.GO), nullable, d.TableIn(elem),
			)
		}
		return fmt.Sprintf("tableimpl.UnmarshalList(%t, %s)", nullable, d.tableElemUnmarshal(elem))
	case t.Unmarshaler != nil:
		fn := "tableimpl.UnmarshalFunc"
		if t.IsContext {
			fn = "tableimpl.UnmarshalFuncContext"
		}
		if isPtr {
			return fmt.Sprintf(
				"%sPtr[*executionContext](%t, %s)",
				fn,
				nullable,
				templates.Call(t.Unmarshaler),
			)
		}
		return fmt.Sprintf("%s[*executionContext](%s)", fn, templates.Call(t.Unmarshaler))
	case t.IsMarshaler:
		if isPtr {
			return fmt.Sprintf(
				"tableimpl.UnmarshalGQLPtr[*executionContext, %s](%t)",
				tableType(ptr.Elem()),
				nullable,
			)
		}
		return fmt.Sprintf("tableimpl.UnmarshalGQL[*executionContext, %s]()", tableType(t.GO))
	case t.Definition.Kind == ast.InputObject:
		if isPtr {
			return fmt.Sprintf(
				"tableimpl.UnmarshalInputPtr[*executionContext, %s](%t, &t.input%s)",
				tableType(t.GO), nullable, t.Definition.Name,
			)
		}
		return fmt.Sprintf(
			"tableimpl.UnmarshalInput[*executionContext, %s](&t.input%s)",
			tableType(t.GO),
			t.Definition.Name,
		)
	}
	panic("codegen: no typed unmarshaler for " + t.UnmarshalFunc())
}

// tableUnmarshalErased is tableMarshalErased for the tableimpl.In of t.
func (d *Data) tableUnmarshalErased(t *config.TypeReference) string {
	nullable := tableNullable(t)
	switch {
	case t.IsSlice():
		elem := t.Elem()
		if d.tableUnmarshalKind(elem) == tableKindErased && elem.IsNilable() {
			return fmt.Sprintf(
				"tableimpl.ListInOf[%s](%t, %s)",
				tableType(elem.GO), nullable, d.TableIn(elem),
			)
		}
		return fmt.Sprintf("tableimpl.ListIn(%t, %s)", nullable, d.tableElemUnmarshal(elem))
	case tableCast(t):
		if t.IsPtr() {
			return fmt.Sprintf(
				"tableimpl.CastPtrIn[*executionContext](%t, %s, (%s)(nil))",
				nullable, templates.Call(t.Unmarshaler), tableType(t.GO),
			)
		}
		return fmt.Sprintf(
			"tableimpl.CastIn[*executionContext](%s, (*%s)(nil))",
			templates.Call(t.Unmarshaler),
			tableType(t.GO),
		)
	case t.IsMarshaler:
		if t.IsPtr() {
			return fmt.Sprintf(
				"tableimpl.GQLPtrIn[*executionContext](%t, (%s)(nil))",
				nullable,
				tableType(t.GO),
			)
		}
		return fmt.Sprintf(
			"tableimpl.GQLIn[*executionContext](%t, (*%s)(nil))",
			nullable,
			tableType(t.GO),
		)
	case t.Definition.Kind == ast.InputObject:
		if t.IsPtr() {
			return fmt.Sprintf(
				"tableimpl.InputPtrIn(t.schema, %t, &t.input%s)",
				nullable,
				t.Definition.Name,
			)
		}
		return fmt.Sprintf("tableimpl.InputIn(t.schema, &t.input%s)", t.Definition.Name)
	}
	panic("codegen: no erased unmarshaler for " + t.UnmarshalFunc())
}

// tableElemUnmarshal is tableElemMarshal for the unmarshaler of the element of a list.
func (d *Data) tableElemUnmarshal(elem *config.TypeReference) string {
	switch d.tableUnmarshalKind(elem) {
	case tableKindTyped:
		return "t." + d.tableName(elem.UnmarshalFunc()) + ".Unmarshal"
	case tableKindErased:
		return d.tableUnmarshalTyped(elem)
	}
	return elem.UnmarshalFunc()
}

// tableCombinable reports whether the runtime may marshal and unmarshal t in table mode.
// Casts to other types than string, lists of casts, enums and pointers to slices,
// interfaces and pointers keep the functions that the functions mode generates. A type
// that marshals itself is bound through its methods, whatever the model of its scalar.
func (d *Data) tableCombinable(t *config.TypeReference) bool {
	if t.CastType != nil && !t.IsMarshaler && (!tableCast(t) || t.IsSlice()) {
		return false
	}
	return !t.HasEnumValues() && !t.IsPtrToSlice() && !t.IsPtrToIntf() && !t.IsPtrToPtr()
}

// tableCheckDirectives returns an error for a directive that the functions mode runs on a
// field, an argument or an input object, but that the tables do not declare: one that a
// plugin declared outside the schema files of the configuration, which the functions mode
// does not compile and the runtime would not run. The inputs that unmarshal themselves are
// left out, as neither mode runs the directives of their fields.
func (d *Data) tableCheckDirectives() error {
	declared := map[string]bool{}
	for _, dir := range d.TableDirectiveDefs() {
		declared[dir.Name] = true
	}
	check := func(dirs []*codegen.Directive, where string) error {
		for _, dir := range dirs {
			if !declared[dir.Name] {
				return fmt.Errorf(
					"table mode cannot run @%s on %s, which is declared outside the schema files of the configuration",
					dir.Name,
					where,
				)
			}
		}
		return nil
	}
	for _, o := range slices.Concat(d.Objects, d.Inputs) {
		if o.IsInputType() && o.HasUnmarshal() {
			continue
		}
		if err := check(o.InputObjectDirectives(), o.Name); err != nil {
			return err
		}
		for _, f := range o.Fields {
			if err := check(f.ImplDirectives(), o.Name+"."+f.Name); err != nil {
				return err
			}
			for _, a := range f.Args {
				if err := check(a.ImplDirectives(), o.Name+"."+f.Name+"("+a.Name+":)"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// tableState returns what the runtime reads from the schema, as d holds it for the
// functions mode, by the element it is about: the directives that the functions mode runs
// on the fields, arguments and inputs, whether it runs those of absent arguments, the
// types of the fields, arguments and input fields, the defaults of the input fields,
// and whether an object is a root, implements interfaces or resolves its fields one after
// another. It also holds what decides the schema that the runtime reads: the sources as
// the generated code holds them, the linked schema, and the directives that the tables
// declare. Build keeps it, and the generation fails where a plugin changes it in d
// afterwards: one mode would follow the change, and the other not. The inputs that
// unmarshal themselves are left out, as neither mode reads their fields, and so are the
// objects that no field returns, which neither mode resolves, such as the type Entity that
// the federation plugin makes a root in its GenerateCode.
func (d *Data) tableState() map[string]string {
	dirs := func(list []*codegen.Directive) string {
		var b strings.Builder
		for _, dir := range list {
			fmt.Fprintf(&b, " @%s(", dir.Name)
			for _, a := range dir.Args {
				fmt.Fprintf(&b, "%s: %#v %#v, ", a.Name, a.Value, a.Default)
			}
			b.WriteString(")")
		}
		return b.String()
	}
	gqlType := func(ref *config.TypeReference, def *ast.Type) string {
		if ref != nil {
			return ref.GQL.String()
		}
		return def.String()
	}
	state := map[string]string{
		"call_argument_directives_with_null": strconv.FormatBool(
			d.Config.CallArgumentDirectivesWithNull,
		),
		"the linked schema": d.TableLinkedSchema,
	}
	var sources strings.Builder
	for _, src := range d.AugmentedSources {
		input := src.Source
		if src.Embeddable {
			b, err := os.ReadFile(
				filepath.Join(d.Config.Exec.Dir(), filepath.FromSlash(src.RelativePath)),
			)
			if err == nil {
				input = string(b)
			}
		}
		fmt.Fprintf(&sources, "%s builtin=%t\n%s\n", src.RelativePath, src.BuiltIn, input)
	}
	state["the sources"] = sources.String()
	for _, dir := range d.TableDirectiveDefs() {
		state["directive @"+dir.Name] = ""
	}
	returned := d.tableReturned()
	for _, o := range d.Objects {
		if !returned[o.Name] {
			continue
		}
		var implements []string
		for _, i := range o.Implements {
			implements = append(implements, i.Name)
		}
		state["type "+o.Name] = fmt.Sprintf("root=%t sequential=%t stream=%t implements=%v",
			o.Root, o.DisableConcurrency, o.Stream, implements)
		for _, f := range o.Fields {
			state[o.Name+"."+f.Name] = fmt.Sprintf("%s%s",
				gqlType(f.TypeReference, f.Type), dirs(f.ImplDirectives()))
			for _, a := range f.Args {
				state[o.Name+"."+f.Name+"("+a.Name+":)"] = fmt.Sprintf("%s with_null=%t%s",
					gqlType(a.TypeReference, a.Type), a.CallArgumentDirectivesWithNull,
					dirs(a.ImplDirectives()))
			}
		}
	}
	for _, in := range d.Inputs {
		if in.HasUnmarshal() {
			continue
		}
		state["input "+in.Name] = dirs(in.InputObjectDirectives())
		for _, f := range in.Fields {
			state[in.Name+"."+f.Name] = fmt.Sprintf(
				"%s = %#v%s",
				gqlType(f.TypeReference, f.Type),
				f.Default,
				dirs(f.ImplDirectives()),
			)
		}
	}
	return state
}

// tableReturned returns the names of the root operation types of the schema and of the
// types that a field of the schema returns, with the implementations of the interfaces and
// the members of the unions among them.
func (d *Data) tableReturned() map[string]bool {
	returned := map[string]bool{}
	for _, root := range []*ast.Definition{d.Schema.Query, d.Schema.Mutation, d.Schema.Subscription} {
		if root != nil {
			returned[root.Name] = true
		}
	}
	for _, def := range d.Schema.Types {
		for _, f := range def.Fields {
			name := f.Type.Name()
			if returned[name] {
				continue
			}
			returned[name] = true
			if typ := d.Schema.Types[name]; typ != nil && typ.IsAbstractType() {
				for _, p := range d.Schema.GetPossibleTypes(typ) {
					returned[p.Name] = true
				}
			}
		}
	}
	return returned
}

// tableCheckState returns an error for an element whose tableState a plugin changed since
// Build.
func (d *Data) tableCheckState() error {
	if d.shared.built == nil {
		return nil
	}
	state := d.tableState()
	keys := slices.Collect(maps.Keys(state))
	for key := range d.shared.built {
		if _, ok := state[key]; !ok {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	for _, key := range keys {
		now, ok := state[key]
		built, wasOK := d.shared.built[key]
		if now != built || ok != wasOK {
			return fmt.Errorf(
				"table mode cannot run what a plugin changed on %s in the data of the generated code, as its runtime reads that from the schema; change the schema with a plugin.SchemaMutator, or generate without -tags gqlgen_table",
				key,
			)
		}
	}
	return nil
}

// TableDirectiveDefs returns the directives that table mode declares an
// tableimpl.DirectiveDef for: those of this schema that can be applied to fields, arguments
// and input objects, which are the places the tables run directives. They are in the
// order of their names, so that what is derived from them does not depend on the
// order of a map.
func (d *Data) TableDirectiveDefs() []*codegen.Directive {
	defs := locationDirectives(
		d.Directives(),
		ast.LocationFieldDefinition,
		ast.LocationObject,
		ast.LocationArgumentDefinition,
		ast.LocationInputFieldDefinition,
		ast.LocationInputObject,
	)
	res := make([]*codegen.Directive, 0, len(defs))
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		res = append(res, defs[name])
	}
	return res
}

func (d *Data) tableOutVar(t *config.TypeReference) string {
	return "out" + templates.UcFirst(d.tableName(t.MarshalFunc()))
}

func (d *Data) tableInVar(t *config.TypeReference) string {
	return "in" + templates.UcFirst(d.tableName(t.UnmarshalFunc()))
}

// TableOutTypes returns the types that fields return, once each, for which table mode
// declares an tableimpl.Out.
func (d *Data) TableOutTypes() []*config.TypeReference {
	seen := map[string]bool{}
	var res []*config.TypeReference
	for _, o := range d.Objects {
		for _, f := range o.Fields {
			name := f.TypeReference.MarshalFunc()
			if !seen[name] && d.GeneratesMarshaler(f.TypeReference) {
				seen[name] = true
				res = append(res, f.TypeReference)
			}
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].MarshalFunc() < res[j].MarshalFunc() })
	return res
}

// TableInTypes returns the types that arguments and input fields take, once each, for
// which table mode declares an tableimpl.In.
func (d *Data) TableInTypes() []*config.TypeReference {
	seen := map[string]bool{}
	var res []*config.TypeReference
	add := func(t *config.TypeReference) {
		if name := t.UnmarshalFunc(); !seen[name] && d.GeneratesUnmarshaler(t) {
			seen[name] = true
			res = append(res, t)
		}
	}
	for _, o := range d.Objects {
		for _, f := range o.Fields {
			for _, a := range f.Args {
				add(a.TypeReference)
			}
		}
	}
	for _, in := range d.Inputs {
		if in.HasUnmarshal() {
			continue
		}
		for _, f := range in.Fields {
			add(f.TypeReference)
		}
	}
	for _, dir := range d.TableDirectiveDefs() {
		for _, a := range dir.Args {
			add(a.TypeReference)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].UnmarshalFunc() < res[j].UnmarshalFunc() })
	return res
}

// tableType writes t as the code that builds the tables refers to it. It writes the
// empty interface as any: the versions of Go differ in whether a type of the schema
// keeps the alias any or writes interface{}, and the generated code must not depend on
// the version of Go that generates it.
func tableType(t types.Type) string {
	return strings.ReplaceAll(templates.CurrentImports.LookupType(t), "interface{}", "any")
}

// MarshalCall returns the function that generated functions call to marshal t. In table
// mode a marshaler built from the runtime's combinators is a field of ec.tables, and
// must be typed: tableTypedCalls lists the callers.
func (d *Data) MarshalCall(t *config.TypeReference) string {
	switch d.tableMarshalKind(t) {
	case tableKindTyped:
		return "ec.tables." + d.tableName(t.MarshalFunc()) + ".Marshal"
	case tableKindErased:
		panic("codegen: the typed marshaler of " + t.MarshalFunc() + " is not generated")
	}
	return t.MarshalFunc()
}

// addPluginCall records that code that a plugin generated into the package of the
// executor calls the marshaler or unmarshaler called name, which MarshalCall or
// UnmarshalCall returns. Table mode then keeps it typed, as the function of the functions
// mode or in an tableimpl.OutFor or tableimpl.InFor of the tables, where it would otherwise build it
// without a type parameter, which generated code cannot call. See collectPluginCalls.
func (d *Data) addPluginCall(name string) {
	if d.shared.calledByPlugins == nil {
		d.shared.calledByPlugins = map[string]bool{}
	}
	d.shared.calledByPlugins[name] = true
}

// tableRederive drops what table mode derived from d, with the objects with batch
// resolvers, so that they are derived again from d as it is then: see tableInfo.
func (d *Data) tableRederive() {
	d.info = nil
	d.groups = nil
}

// UnmarshalCall is MarshalCall for the unmarshaler of t.
func (d *Data) UnmarshalCall(t *config.TypeReference) string {
	switch d.tableUnmarshalKind(t) {
	case tableKindTyped:
		return "ec.tables." + d.tableName(t.UnmarshalFunc()) + ".Unmarshal"
	case tableKindErased:
		panic("codegen: the typed unmarshaler of " + t.UnmarshalFunc() + " is not generated")
	}
	return t.UnmarshalFunc()
}

// TableCombinator is a marshaler or unmarshaler that table mode builds from the
// runtime's combinators, as a field of the tables.
type TableCombinator struct {
	// Name is the name of the field, which is the name of the generated function in the
	// functions mode.
	Name string
	// Type is the type reference the combinator marshals or unmarshals.
	Type *config.TypeReference
	// Marshal is set for a marshaler, and unset for an unmarshaler.
	Marshal bool
	// Erased is set when the field holds an tableimpl.Out or tableimpl.In that is not generic in
	// the Go type, rather than an tableimpl.OutFor or tableimpl.InFor.
	Erased bool
	// Expr is the Go expression that builds the field.
	Expr  string
	depth int
}

// TableCombinators returns the marshalers and unmarshalers that table mode builds from
// the runtime, in an order in which each comes after the ones it uses: a list uses the
// marshaler of its elements, which has one list less. Those that the runtime derives are
// left out.
func (d *Data) TableCombinators() []TableCombinator {
	seen := map[string]bool{}
	var res []TableCombinator
	add := func(name string, t *config.TypeReference, marshal bool, kind tableKind, expr string) {
		if name == "" || expr == "" || seen[name] {
			return
		}
		seen[name] = true
		res = append(res, TableCombinator{
			Name:    d.tableName(name),
			Type:    t,
			Marshal: marshal,
			Erased:  kind == tableKindErased,
			Expr:    expr,
			depth:   listDepth(t),
		})
	}
	// The expressions register the imports they use, which name a package that shares
	// its name with another one first, so the types go in a fixed order.
	for _, key := range slices.Sorted(maps.Keys(d.ReferencedTypes)) {
		t := d.ReferencedTypes[key]
		if !d.tableDerived(t, false) {
			add(t.UnmarshalFunc(), t, false, d.tableUnmarshalKind(t), d.TableUnmarshal(t))
		}
		if !d.tableDerived(t, true) {
			add(t.MarshalFunc(), t, true, d.tableMarshalKind(t), d.TableMarshal(t))
		}
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].depth != res[j].depth {
			return res[i].depth < res[j].depth
		}
		return res[i].Name < res[j].Name
	})
	return res
}

// listDepth returns how many lists t is nested in.
func listDepth(t *config.TypeReference) int {
	if !t.IsSlice() {
		return 0
	}
	return 1 + listDepth(t.Elem())
}

// TableObjects returns the objects and inputs that table mode generates a table for.
func (d *Data) TableObjects() []*codegen.Object {
	res := slices.Clone(d.Objects)
	for _, in := range d.Inputs {
		if !in.HasUnmarshal() {
			res = append(res, in)
		}
	}
	return res
}

// TableField is a line of the fields of the tables struct that table mode generates.
type TableField struct {
	// Name holds the names of the fields of the line, separated by commas: fields of the
	// same type share a line, up to tableFieldsPerLine of them.
	Name string
	Type string
	// Section is set on the first line of a run of lines of about the same length. gofmt
	// aligns the types of a run of fields, so the template separates the runs to keep
	// short lines from being padded to the length of long ones.
	Section bool
}

// tableFieldsPerLine is how many fields of the same type share a line of the tables
// struct at most.
const tableFieldsPerLine = 6

// TableFields returns the lines of the fields of the tables struct: the tables of the
// objects, inputs and directives, the combinators they are built from, and the tableimpl.Out
// and tableimpl.In of each type, group by group.
func (d *Data) TableFields() []TableField {
	var res []TableField
	for _, g := range d.tableFieldGroups() {
		res = append(res, tableFieldLines(g)...)
	}
	return res
}

// tableFieldGroups returns the fields of the tables struct, one per line, in groups:
// the tables of the objects and inputs, those of the directives, the shared directive
// lists, the combinators, and the tableimpl.Out and tableimpl.In of the generated functions.
func (d *Data) tableFieldGroups() [][]TableField {
	const ec = "*executionContext"
	var groups [][]TableField
	var group []TableField
	for _, o := range d.TableObjects() {
		kind := "Object"
		if o.Kind == ast.InputObject {
			kind = "Input"
		}
		group = append(group, TableField{
			Name: d.TableObjectVar(o),
			Type: "tableimpl." + kind + "[" + ec + "]",
		})
	}
	groups = append(groups, group)
	group = nil
	for _, dir := range d.TableDirectiveDefs() {
		group = append(group, TableField{
			Name: d.TableDirectiveVar(dir),
			Type: "tableimpl.DirectiveDef[" + ec + "]",
		})
	}
	groups = append(groups, group)
	group = nil
	for _, c := range d.TableCombinators() {
		var typ string
		switch {
		case c.Erased && c.Marshal:
			typ = "tableOut"
		case c.Erased:
			typ = "tableIn"
		case c.Marshal:
			typ = "tableimpl.OutFor[" + ec + ", " + tableType(c.Type.GO) + "]"
		default:
			typ = "tableimpl.InFor[" + ec + ", " + tableType(c.Type.GO) + "]"
		}
		group = append(group, TableField{Name: c.Name, Type: typ})
	}
	groups = append(groups, group)
	group = nil
	for _, t := range d.TableOutTypes() {
		group = append(group, TableField{
			Name: d.tableOutVar(t),
			Type: "tableimpl.OutFor[" + ec + ", " + tableType(t.GO) + "]",
		})
	}
	groups = append(groups, group)
	group = nil
	for _, t := range d.TableInTypes() {
		group = append(group, TableField{
			Name: d.tableInVar(t),
			Type: "tableimpl.InFor[" + ec + ", " + tableType(t.GO) + "]",
		})
	}
	groups = append(groups, group)
	return groups
}

// tableFieldLines lays out the fields of one group of the tables struct. Fields of a
// type of their own come first, by the length of their names; the fields of each type
// that several share follow, type by type, up to tableFieldsPerLine of them per line. A
// line much longer or shorter than the first of its section starts a new section.
func tableFieldLines(g []TableField) []TableField {
	count := map[string]int{}
	for _, f := range g {
		count[f.Type]++
	}
	sort.Slice(g, func(i, j int) bool {
		ti, tj := g[i].Type, g[j].Type
		if count[ti] == 1 {
			ti = ""
		}
		if count[tj] == 1 {
			tj = ""
		}
		if ti != tj {
			return ti < tj
		}
		if len(g[i].Name) != len(g[j].Name) {
			return len(g[i].Name) < len(g[j].Name)
		}
		return g[i].Name < g[j].Name
	})
	var lines []TableField
	for i := 0; i < len(g); {
		j := i + 1
		for j < len(g) && j-i < tableFieldsPerLine && g[j].Type == g[i].Type {
			j++
		}
		names := make([]string, 0, j-i)
		for _, f := range g[i:j] {
			names = append(names, f.Name)
		}
		lines = append(lines, TableField{Name: strings.Join(names, ", "), Type: g[i].Type})
		i = j
	}
	start := 0
	for i := range lines {
		n, s := len(lines[i].Name), len(lines[start].Name)
		if i == 0 || n > s*5/4 || n < s*4/5 {
			lines[i].Section = true
			start = i
		}
	}
	return lines
}

// tableInitChunk is how many marshalers, tableimpl.Out and tableimpl.In one generated function of
// the tables sets. Large schemas have thousands of them, and the compiler takes much
// longer for one function that sets them all than for several that each set some.
const tableInitChunk = 200

// TableInitChunks returns the statements that set the marshalers, unmarshalers,
// tableimpl.Out and tableimpl.In of the tables, in the order in which they must run, split into
// the bodies of several functions.
func (d *Data) TableInitChunks() [][]string {
	var stmts []string
	for _, c := range d.TableCombinators() {
		stmts = append(stmts, "t."+c.Name+" = "+c.Expr)
	}
	for _, t := range d.TableOutTypes() {
		stmts = append(stmts, "t."+d.tableOutVar(t)+" = tableimpl.OutOf("+t.MarshalFunc()+")")
	}
	for _, t := range d.TableInTypes() {
		stmts = append(stmts, "t."+d.tableInVar(t)+" = tableimpl.InOf("+t.UnmarshalFunc()+")")
	}
	var res [][]string
	for chunk := range slices.Chunk(stmts, tableInitChunk) {
		res = append(res, chunk)
	}
	return res
}

// tableInitFields is how many fields, of objects or inputs, one generated function sets
// at most, unless a single table has more. The compiler needs more memory for larger
// functions, so the tables are not all built by one function, but most tables have a
// few fields, and one function per table costs lines.
const tableInitFields = 16

// TableInitGroups returns the groups of tables that the generated file builds, each by
// one generated function named after its first table: those of the file's own types
// for a file of the follow-schema layout, and all groups of the schema for the root
// file, which calls the function of every group, and for the single-file layout.
func (d *Data) TableInitGroups() [][]*codegen.Object {
	d.tableInfo()
	return d.groups
}

// tableInitGroups returns the groups of tables of the whole schema, by the file that
// builds them in the follow-schema layout and in the order of the files, and all of
// them in that order.
func (d *Data) tableInitGroups() (all [][]*codegen.Object, byFile map[string][][]*codegen.Object) {
	tables := map[string][]*codegen.Object{}
	var files []string
	for _, o := range d.TableObjects() {
		var file string
		if d.Config.Exec.Layout == config.ExecLayoutFollowSchema {
			file = filename(o.Position, d.Config)
		}
		if _, ok := tables[file]; !ok {
			files = append(files, file)
		}
		tables[file] = append(tables[file], o)
	}
	byFile = make(map[string][][]*codegen.Object, len(files))
	for _, file := range files {
		groups := groupTables(tables[file])
		byFile[file] = groups
		all = append(all, groups...)
	}
	return all, byFile
}

// groupTables splits the tables of one file into groups of at most tableInitFields
// fields, unless a single table has more, each of which one generated function builds.
func groupTables(tables []*codegen.Object) [][]*codegen.Object {
	var res [][]*codegen.Object
	var group []*codegen.Object
	fields := 0
	for _, o := range tables {
		if len(group) > 0 && fields+len(o.Fields) > tableInitFields {
			res = append(res, group)
			group, fields = nil, 0
		}
		group = append(group, o)
		fields += len(o.Fields)
	}
	if len(group) > 0 {
		res = append(res, group)
	}
	return res
}

// TableOut returns how the code that builds the tables refers to the tableimpl.Out of t: the
// field that holds it, or the Out of the tableimpl.OutFor of its combinator or of its generated
// function.
func (d *Data) TableOut(t *config.TypeReference) string {
	if d.tableDerived(t, true) {
		panic("codegen: the Out of " + t.MarshalFunc() + " is derived by the runtime")
	}
	switch d.tableMarshalKind(t) {
	case tableKindTyped:
		return "t." + d.tableName(t.MarshalFunc()) + ".Out"
	case tableKindErased:
		return "t." + d.tableName(t.MarshalFunc())
	}
	return "t." + d.tableOutVar(t) + ".Out"
}

// TableOutEntry returns the entry of a field of type t in the table of its object that
// says how the field is marshaled: its Out, or its Go type when the runtime derives the
// Out from the type of the field. See tableDerived.
func (d *Data) TableOutEntry(t *config.TypeReference) string {
	if d.tableDerived(t, true) {
		return "Type: " + tableZero(t)
	}
	return "Out: " + d.TableOut(t)
}

// TableInEntry is TableOutEntry for an input field, which has an In.
func (d *Data) TableInEntry(t *config.TypeReference) string {
	if d.tableDerived(t, false) {
		return "Type: " + tableZero(t)
	}
	return "In: " + d.TableIn(t)
}

// tableZero returns the nil pointer to the Go type of t that gives the runtime the type
// to derive the Out or In of t from.
func tableZero(t *config.TypeReference) string {
	return "(*" + tableType(t.GO) + ")(nil)"
}

// tableDerived reports whether the runtime derives the tableimpl.Out of t, or its tableimpl.In when
// marshal is unset, from the GraphQL type and the Go type alone, so that the tables
// declare neither: when table mode builds it with a constructor that is not generic in
// the Go type, for the pointers to an object, the scalars and enums that marshal
// themselves, the input objects, and the lists of those, unless the expression of
// another marshaler or unmarshaler of the tables refers to it. See tableNoDerive.
func (d *Data) tableDerived(t *config.TypeReference, marshal bool) bool {
	info := d.tableInfo()
	no := info.noDeriveIn
	if marshal {
		no = info.noDeriveOut
	}
	return d.tableDerives(t, marshal, no)
}

// tableDerives is tableDerived with the type references that stay declared in no.
func (d *Data) tableDerives(t *config.TypeReference, marshal bool, no map[string]bool) bool {
	name, kind := t.UnmarshalFunc(), d.tableUnmarshalKind(t)
	if marshal {
		name, kind = t.MarshalFunc(), d.tableMarshalKind(t)
	}
	if kind != tableKindErased || no[name] {
		return false
	}
	switch defKind := t.Definition.Kind; {
	case t.IsSlice():
		// Nested lists keep their marshalers: the outer list calls the typed marshaler
		// of the inner one.
		elem := t.Elem()
		return !elem.IsSlice() && d.tableDerives(elem, marshal, no)
	case tableCast(t):
		// A cast needs the marshaler of string.
		return false
	case t.IsMarshaler:
		return defKind == ast.Scalar || defKind == ast.Enum
	case marshal:
		return defKind == ast.Object
	default:
		return defKind == ast.InputObject
	}
}

// tableNoDerive returns the marshalers and unmarshalers that the runtime could derive
// but that stay declared, because a list that the tables declare refers to them: a list
// whose elements are passed to their Out or In as any. Such a list is declared when the
// generated code calls its typed marshaler or unmarshaler.
func (d *Data) tableNoDerive() (out, in map[string]bool) {
	out, in = map[string]bool{}, map[string]bool{}
	for changed := true; changed; {
		changed = false
		for _, t := range d.ReferencedTypes {
			if !t.IsSlice() {
				continue
			}
			elem := t.Elem()
			if d.tableMarshalKind(t) != tableKindFunc && !d.tableDerives(t, true, out) &&
				d.tableMarshalKind(elem) == tableKindErased && elem.IsNilable() &&
				!out[elem.MarshalFunc()] {
				out[elem.MarshalFunc()] = true
				changed = true
			}
			if d.tableUnmarshalKind(t) != tableKindFunc && !d.tableDerives(t, false, in) &&
				d.tableUnmarshalKind(elem) == tableKindErased && elem.IsNilable() &&
				!in[elem.UnmarshalFunc()] {
				in[elem.UnmarshalFunc()] = true
				changed = true
			}
		}
	}
	return out, in
}

// TableIn is TableOut for the tableimpl.In of t.
func (d *Data) TableIn(t *config.TypeReference) string {
	if d.tableDerived(t, false) {
		panic("codegen: the In of " + t.UnmarshalFunc() + " is derived by the runtime")
	}
	switch d.tableUnmarshalKind(t) {
	case tableKindTyped:
		return "t." + d.tableName(t.UnmarshalFunc()) + ".In"
	case tableKindErased:
		return "t." + d.tableName(t.UnmarshalFunc())
	}
	return "t." + d.tableInVar(t) + ".In"
}

// tableTypedCalls returns the marshalers and unmarshalers whose typed form the generated
// code calls, which keep an tableimpl.OutFor or tableimpl.InFor even when their tableimpl.Out or tableimpl.In
// could be built without a type parameter: those of the elements of the type references
// the combinators do not cover, which the functions generated for them call, and those of
// the arguments of directives, which the functions that parse the arguments of the
// directives of queries call, the marshalers and unmarshalers that plugins call (see
// addPluginCall), and the inner lists of nested lists, which the outer list
// marshals with a typed marshaler: through an Out, each inner slice would be boxed in an
// any. The elements of lists do not count otherwise: the lists take the combinator of an
// erased element, or its Out or In.
func (d *Data) tableTypedCalls() map[string]bool {
	typed := map[string]bool{}
	for _, t := range d.ReferencedTypes {
		// The generated functions of these shapes call the marshaler and unmarshaler of
		// their element; see type.gotpl.
		elem := t.Elem()
		callsElem := t.IsSlice() || t.IsPtrToSlice() || t.IsPtrToIntf() ||
			(t.IsPtrToPtr() && t.Unmarshaler == nil && !t.IsMarshaler)
		if elem == nil || !callsElem {
			continue
		}
		if t.IsSlice() && elem.IsSlice() {
			typed[elem.MarshalFunc()] = true
			typed[elem.UnmarshalFunc()] = true
		}
		if d.tableMarshalBase(t) == tableKindFunc {
			typed[elem.MarshalFunc()] = true
		}
		if d.tableUnmarshalBase(t) == tableKindFunc {
			typed[elem.UnmarshalFunc()] = true
		}
	}
	for _, dir := range d.Directives() {
		for _, a := range dir.Args {
			typed[a.TypeReference.UnmarshalFunc()] = true
		}
	}
	for name := range d.shared.calledByPlugins {
		typed[name] = true
	}
	return typed
}

// shortTypeIdentifier returns id, the identifier of a Go type that
// templates.TypeIdentifier returns, without the import path of the type's package but
// for its last element: ᚖmodelᚐUser for ᚖgithubᚗcomᚋuserᚋappᚋmodelᚐUser.
func shortTypeIdentifier(id string) string {
	pkgEnd := strings.LastIndex(id, "ᚐ")
	if pkgEnd < 0 {
		return id
	}
	lastSlash := strings.LastIndex(id[:pkgEnd], "ᚋ")
	if lastSlash < 0 {
		return id
	}
	pathStart := len(id) - len(strings.TrimLeft(id, "ᚖᚕ"))
	return id[:pathStart] + id[lastSlash+len("ᚋ"):]
}

// tableInfo is what table mode derives from the whole schema once. The files of the
// follow-schema layout each hold some of the types, but the names of the tables and the
// functions that build them follow from all of them, so the Data of the root file builds
// it and hands it to the Data of each file.
type tableInfo struct {
	// names maps the marshalers and unmarshalers of the schema to the names of the
	// fields of the tables. See tableName.
	names map[string]string
	// groups are the groups of tables that one generated function each builds, in the
	// order of the files, and groupsByFile holds those of each file.
	groups       [][]*codegen.Object
	groupsByFile map[string][][]*codegen.Object
	// typedCalls holds the marshalers and unmarshalers that the generated code calls,
	// which keep their typed form. See tableTypedCalls.
	typedCalls map[string]bool
	// noDeriveOut and noDeriveIn hold the marshalers and unmarshalers that the tables
	// declare although the runtime could derive them. See tableNoDerive.
	noDeriveOut, noDeriveIn map[string]bool
	// goTypes holds the Go types of the objects, inputs, interfaces and unions by name: the
	// first of their models, which the functions mode marshals and unmarshals. See
	// tableOwnType.
	goTypes map[string]types.Type
	// unmarshalsItself holds the inputs that unmarshal themselves, which have no table.
	unmarshalsItself map[string]bool
	// batchParents holds the objects that resolve a field through a batch resolver, and
	// the interfaces and unions that one of them implements, by name. See tableBatchList.
	batchParents map[string]bool
}

// tableInfo returns what table mode derives from the whole schema, building it on the
// first call. A Data made for one file of the follow-schema layout is given that of the
// root file instead.
func (d *Data) tableInfo() *tableInfo {
	if d.info == nil {
		// The kinds, which the rest follows from, read the Go types of the objects and
		// inputs.
		d.info = &tableInfo{goTypes: map[string]types.Type{}, unmarshalsItself: map[string]bool{}}
		for _, o := range slices.Concat(d.Objects, d.Inputs) {
			d.info.goTypes[o.Name] = o.Type
		}
		for _, in := range d.Inputs {
			d.info.unmarshalsItself[in.Name] = in.HasUnmarshal()
		}
		for name, i := range d.Interfaces {
			d.info.goTypes[name] = i.Type
		}
		// The objects with batch resolvers are those of the data as it is now, which a
		// plugin may have changed since codegen.Data computed them.
		batchFieldTypes := collectBatchFieldTypes(d.Objects)
		d.info.batchParents = map[string]bool{}
		for name := range batchFieldTypes {
			d.info.batchParents[name] = true
		}
		for name := range d.Interfaces {
			if len(batchImplementors(d.Interfaces, batchFieldTypes, name)) > 0 {
				d.info.batchParents[name] = true
			}
		}
		d.info.names = d.tableNames()
		d.info.typedCalls = d.tableTypedCalls()
		d.info.noDeriveOut, d.info.noDeriveIn = d.tableNoDerive()
		d.info.groups, d.info.groupsByFile = d.tableInitGroups()
		d.groups = d.info.groups
	}
	return d.info
}

// tableName returns the name of the field of the tables for the marshaler or unmarshaler
// called name: the name without the import paths of its packages, such as
// marshalNUser2modelᚐUser, unless another name of the schema shortens to the same.
func (d *Data) tableName(name string) string {
	if short, ok := d.tableInfo().names[name]; ok {
		return short
	}
	return name
}

// tableNames returns the short names of the marshalers and unmarshalers of the schema,
// which tableName uses.
func (d *Data) tableNames() map[string]string {
	full := map[string][]string{}
	for _, t := range d.ReferencedTypes {
		id := templates.TypeIdentifier(t.GO)
		shortID := shortTypeIdentifier(id)
		for _, name := range []string{t.MarshalFunc(), t.UnmarshalFunc()} {
			if name == "" {
				continue
			}
			short := strings.Replace(name, "2"+id, "2"+shortID, 1)
			if !slices.Contains(full[short], name) {
				full[short] = append(full[short], name)
			}
		}
	}
	names := map[string]string{}
	for short, fullNames := range full {
		for _, name := range fullNames {
			names[name] = name
			if len(fullNames) == 1 {
				names[name] = short
			}
		}
	}
	return names
}

// TableFieldArgs returns the Go expression of the arguments of f for the tables, which
// the runtime matches with the schema by name: first those bound to Go, in the order in
// which the functions mode unmarshals them, which is the order of the parameters of the
// method of the field, with the tableimpl.In of each, or its Go type when the runtime derives
// the In. The others that the schema declares, such as those that the method does not
// take, follow with their names only: the runtime leaves them out, as the functions mode
// leaves them out of the arguments function of the field.
func (d *Data) TableFieldArgs(f *codegen.Field) string {
	args := make([]string, 0, len(f.Arguments))
	for _, arg := range f.Args {
		args = append(args, d.tableArg(arg))
	}
	for _, decl := range f.Arguments {
		named := func(arg *codegen.FieldArgument) bool { return arg.Name == decl.Name }
		if !slices.ContainsFunc(f.Args, named) {
			args = append(args, "{Name: "+strconv.Quote(decl.Name)+"}")
		}
	}
	return "[]tableArg{" + strings.Join(args, ", ") + "}"
}

// tableArg returns the entry of the tables of arg, an argument bound to Go.
func (d *Data) tableArg(arg *codegen.FieldArgument) string {
	var res string
	if d.tableDerived(arg.TypeReference, false) {
		res = "Type: " + tableZero(arg.TypeReference)
	} else {
		res = "In: " + d.TableIn(arg.TypeReference)
	}
	// The errors of the directives of the argument name its Go type.
	if goType := tableGoType(arg.TypeReference); goType != "" && len(arg.ImplDirectives()) > 0 {
		res += ", GoType: " + strconv.Quote(goType)
	}
	return "{Name: " + strconv.Quote(arg.Name) + ", " + res + "}"
}

// TableInputGoType returns the Go type of the input field f for the tables, which the
// errors of its directives name, or "" when the runtime writes it the same. See
// tableGoType.
func (d *Data) TableInputGoType(f *codegen.Field) string {
	if len(f.ImplDirectives()) == 0 {
		return ""
	}
	return tableGoType(f.TypeReference)
}

// tableGoType returns the Go type of t as the functions mode writes it in errors, when
// the runtime writes it otherwise from its reflect.Type, and "" when the two are the same.
// See tableReflectName.
func tableGoType(t *config.TypeReference) string {
	if name := t.GO.String(); name != tableReflectName(t.GO) {
		return name
	}
	return ""
}

// tableReflectName returns t as the runtime writes the reflect.Type of t in errors, with
// the full import path of named types: reflect does not tell byte and rune from uint8 and
// int32, nor interface{} from any, which go/types writes as the source does. It returns
// "" for the types it does not know how the runtime writes, such as instantiated generic
// types.
func tableReflectName(t types.Type) string {
	switch t := t.(type) {
	case *types.Alias:
		return tableReflectName(types.Unalias(t))
	case *types.Named:
		if t.TypeArgs().Len() > 0 {
			return ""
		}
		if t.Obj().Pkg() == nil {
			return t.Obj().Name()
		}
		return t.Obj().Pkg().Path() + "." + t.Obj().Name()
	case *types.Basic:
		return types.Typ[t.Kind()].Name()
	case *types.Interface:
		if t.Empty() {
			return "any"
		}
		return ""
	}
	var elem types.Type
	var format string
	switch t := t.(type) {
	case *types.Pointer:
		elem, format = t.Elem(), "*%s"
	case *types.Slice:
		elem, format = t.Elem(), "[]%s"
	case *types.Array:
		elem, format = t.Elem(), "["+strconv.FormatInt(t.Len(), 10)+"]%s"
	case *types.Map:
		key := tableReflectName(t.Key())
		if key == "" {
			return ""
		}
		elem, format = t.Elem(), "map["+key+"]%s"
	default:
		return ""
	}
	name := tableReflectName(elem)
	if name == "" {
		return ""
	}
	return fmt.Sprintf(format, name)
}

// TableDirectiveArgs is TableFieldArgs for the arguments of a directive, which are all
// bound to Go and unmarshaled by their tableimpl.In, or nil for a directive without
// arguments.
func (d *Data) TableDirectiveArgs(dir *codegen.Directive) string {
	if len(dir.Args) == 0 {
		return "nil"
	}
	args := make([]string, len(dir.Args))
	for i, arg := range dir.Args {
		args[i] = fmt.Sprintf("{Name: %q, In: %s}", arg.Name, d.TableIn(arg.TypeReference))
	}
	return "[]tableArg{" + strings.Join(args, ", ") + "}"
}

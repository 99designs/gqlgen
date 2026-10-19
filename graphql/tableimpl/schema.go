package tableimpl

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/vektah/gqlparser/v2/ast"
)

// Schema links the tables of an executable schema to the schema they execute. The tables
// declare what ties the schema to Go: the getter or resolver of a field, the marshaler
// or unmarshaler of a value, the implementation of a directive. What the schema says
// itself, such as the type of a field, the directives applied to it or the default of
// an argument, Link reads from the schema, so that the generated code does not repeat
// it.
//
// Generated code creates one Schema per executable schema, registers its tables with
// Init and calls Link once.
type Schema[EC Context] struct {
	schema     *ast.Schema
	objects    map[string]*Object[EC]
	inputs     map[string]*Input[EC]
	directives map[string]*DirectiveDef[EC]
	// inputIns are the Ins that InputIn returned, which Link completes from the Go types
	// of their inputs.
	inputIns []inputIn[EC]
	// outs and ins hold the Outs and Ins that Link derives, which the type references of
	// the same GraphQL type and Go type share.
	outs map[typeKey]*Out[EC]
	ins  map[typeKey]*In[EC]

	// OmitPanicHandler is set when the generated code was configured not to recover
	// panics in resolvers.
	OmitPanicHandler bool
	// WorkerLimit is passed to graphql.MarshalSliceConcurrently by the Outs of lists
	// that Link derives.
	WorkerLimit int64
	// ArgumentDirectivesWithNull runs the directives of an argument even when the
	// argument is absent, as call_argument_directives_with_null does.
	ArgumentDirectivesWithNull bool
}

// NewSchema returns the Schema that the tables of an executable schema of s register
// with.
func NewSchema[EC Context](s *ast.Schema) *Schema[EC] {
	return &Schema[EC]{
		schema:     s,
		objects:    map[string]*Object[EC]{},
		inputs:     map[string]*Input[EC]{},
		directives: map[string]*DirectiveDef[EC]{},
		outs:       map[typeKey]*Out[EC]{},
		ins:        map[typeKey]*In[EC]{},
	}
}

// Link reads from the schema what the registered tables leave to it. Generated code
// calls it once every table is registered, as fields refer to the tables of the types
// they return. It panics when a table does not match the schema, which means that the
// code was generated for another schema.
func (s *Schema[EC]) Link() {
	// The Ins of input objects come first: the arguments and input fields of the others
	// read them.
	for _, x := range s.inputIns {
		x.link()
	}
	// The directives come next: the others need their locations.
	for name, d := range s.directives {
		d.link(s, name)
	}
	for name, o := range s.objects {
		o.link(s, name)
	}
	for name, in := range s.inputs {
		in.link(s, name)
	}
	// The fields hold what Link derived, which nothing looks up any more, and nothing
	// reads the schema after Link, which may be one loaded only to link the tables.
	s.outs, s.ins, s.schema = nil, nil, nil
}

func (s *Schema[EC]) definition(name string) *ast.Definition {
	def := s.schema.Types[name]
	if def == nil {
		panic("tableimpl:the schema has no type " + strconv.Quote(name))
	}
	return def
}

// isRoot reports whether def is a root operation type.
func (s *Schema[EC]) isRoot(def *ast.Definition) bool {
	return def == s.schema.Query || def == s.schema.Mutation || def == s.schema.Subscription
}

// applied returns the directives in dirs that the tables implement, with the argument
// values written in the schema or their defaults. The others are left to the
// generator by skip_runtime, or apply to a place the tables run no directives at.
// For the directives of a type definition, which run around the fields that return the
// type or the input fields of the type, field is the location of those fields,
// FIELD_DEFINITION or INPUT_FIELD_DEFINITION: those that apply to input objects do not
// run there, and only those that apply to the location or to object types do, as in the
// functions mode. field is empty for the directives applied to the element itself.
func (s *Schema[EC]) applied(dirs ast.DirectiveList, field ast.DirectiveLocation) []directive[EC] {
	var res []directive[EC]
	for _, d := range dirs {
		def := s.directives[d.Name]
		if def == nil {
			continue
		}
		if field != "" && (def.declares(ast.LocationInputObject) ||
			!def.declares(field, ast.LocationObject)) {
			continue
		}
		res = append(res, directive[EC]{def: def, args: s.literalArgs(def, d)})
	}
	return res
}

// literalArgs returns the argument values of the directive application d, written in
// the schema or defaulted by the declaration of the directive. Arguments with neither
// are absent, and so are those that are null: the functions mode passes nil for them
// without unmarshaling null, which an argument of a non-null type would fail or turn into
// an empty value. An argument written twice, which the validation of the schema lets
// through, takes the last value, as in the functions mode.
func (s *Schema[EC]) literalArgs(def *DirectiveDef[EC], d *ast.Directive) map[string]any {
	var args map[string]any
	for _, decl := range s.schema.Directives[def.Name].Arguments {
		v := decl.DefaultValue
		for _, a := range d.Arguments {
			if a.Name == decl.Name {
				v = a.Value
			}
		}
		if v == nil || v.Kind == ast.NullValue {
			continue
		}
		if args == nil {
			args = map[string]any{}
		}
		args[decl.Name] = literal(v, "argument "+decl.Name+" of @"+def.Name)
	}
	return args
}

// args returns the arguments of the tables, in their order, with what decls, the
// arguments the schema declares, says about them. The tables match the arguments with
// decls by name, as the generated code reads them by name: they may list them in another
// order, but name every argument that decls declares, and no other. One may come twice,
// for a method that takes one argument in two parameters whose names differ in case
// only, which the functions mode unmarshals twice. An argument that is not bound to Go is
// left out.
func (s *Schema[EC]) args(decls ast.ArgumentDefinitionList, table []Arg[EC], of string) []arg[EC] {
	for _, decl := range decls {
		if !slices.ContainsFunc(table, func(a Arg[EC]) bool { return a.Name == decl.Name }) {
			panic(fmt.Sprintf("tableimpl:the tables have no argument %s of %s", decl.Name, of))
		}
	}
	res := make([]arg[EC], 0, len(table))
	for _, a := range table {
		decl := decls.ForName(a.Name)
		if decl == nil {
			panic(fmt.Sprintf("tableimpl:%s has no argument %s", of, a.Name))
		}
		in := a.In
		if in == nil {
			if a.Type == nil {
				continue
			}
			goType, ok := goType(a.Type)
			if !ok {
				panic(notGoType("argument "+decl.Name+" of "+of, a.Type))
			}
			in = s.derivedIn(decl.Type, goType)
		}
		res = append(res, arg[EC]{
			name:       decl.Name,
			typ:        in,
			directives: s.applied(decl.Directives, ""),
			nilOK:      in.nilable,
			withNull:   s.ArgumentDirectivesWithNull,
			goType:     a.GoType,
		})
	}
	return res
}

// literal returns the Go value of a value written in the schema, as the functions mode
// writes it into its generated code: see asGenerated.
func literal(v *ast.Value, of string) any {
	val, err := v.Value(nil)
	if err != nil {
		panic("tableimpl:invalid value of " + of + ": " + err.Error())
	}
	return asGenerated(val)
}

// asGenerated returns v, a value that the parser produced, as the functions mode writes
// the same literal into its generated code: integers are int, not the int64 of the parser,
// and floats keep six decimals, as the generator formats them with %f. -0 is 0, as the
// generated code writes it as a Go constant, which has no negative zero. An integer that
// int cannot hold, on a 32-bit platform, panics, where the functions mode does not compile.
func asGenerated(v any) any {
	switch v := v.(type) {
	case int64:
		if int64(int(v)) != v {
			panic(fmt.Sprintf("tableimpl:the schema holds the integer %d, which overflows int", v))
		}
		return int(v)
	case float64:
		f, err := strconv.ParseFloat(fmt.Sprintf("%f", v), 64)
		if err != nil {
			return v
		}
		if f == 0 {
			return 0.0
		}
		return f
	case []any:
		for i := range v {
			v[i] = asGenerated(v[i])
		}
	case map[string]any:
		for k := range v {
			v[k] = asGenerated(v[k])
		}
	}
	return v
}

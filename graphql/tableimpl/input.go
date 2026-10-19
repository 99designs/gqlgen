package tableimpl

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strconv"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

// Unmarshal converts a raw input value into V. Generated code passes its unmarshal
// functions.
type Unmarshal[EC, V any] func(ctx context.Context, ec EC, v any) (V, error)

// In unmarshals raw input values into one Go type, returned as any. Generated code
// declares one per type that arguments and input fields take, with InOf.
type In[EC any] struct {
	unmarshal func(ctx context.Context, ec EC, v any) (any, error)
	accept    func(v any) bool
	zero      any
	name      string
	// nilable is set when the Go type can be nil, so that a directive may return nil
	// for a value of it.
	nilable bool
}

// InFor is the In of the values of type V, with the Unmarshal of the type, which the
// generated code calls where it unmarshals values of the type itself: for the elements of
// lists, the arguments of the directives of queries, and in the functions it generates
// for the type references the combinators do not cover.
type InFor[EC, V any] struct {
	*In[EC]
	Unmarshal Unmarshal[EC, V]
}

// InOf returns the In of the values that u unmarshals.
//
//go:noinline
func InOf[EC, V any](u Unmarshal[EC, V]) InFor[EC, V] {
	var zero V
	typ := reflect.TypeFor[V]()
	return InFor[EC, V]{In: &In[EC]{
		unmarshal: func(ctx context.Context, ec EC, v any) (any, error) {
			return u(ctx, ec, v)
		},
		accept: func(v any) bool {
			_, ok := v.(V)
			return ok
		},
		zero:    zero,
		name:    typeString(typ),
		nilable: nilable(typ),
	}, Unmarshal: u}
}

// InputIn returns the In of an input object that is unmarshaled by value into the Go
// type of in, for the arguments and input fields of that type. Unlike InOf, it is not
// generic in the Go type, so that the inputs of a schema share it rather than each
// instantiating it: Schema.Link completes the In from the Go type that the table of in
// records, which is why the In is empty until then. Generated code uses it for the inputs
// whose unmarshaler it does not call itself, when a list of its tables refers to the In;
// Link derives the same In for the others. Those it calls use UnmarshalInput.
//
//go:noinline
func InputIn[EC Context](s *Schema[EC], in *Input[EC]) *In[EC] {
	res := &In[EC]{}
	s.inputIns = append(s.inputIns, inputIn[EC]{in: res, input: in})
	return res
}

// InputPtrIn is InputIn for a pointer to the Go type of in, which unmarshals as
// UnmarshalInputPtr does. Nullable references unmarshal null as nil.
//
//go:noinline
func InputPtrIn[EC Context](s *Schema[EC], nullable bool, in *Input[EC]) *In[EC] {
	res := &In[EC]{}
	s.inputIns = append(s.inputIns, inputIn[EC]{
		in:       res,
		input:    in,
		pointer:  true,
		nullable: nullable,
	})
	return res
}

// inputIn is an In that InputIn or InputPtrIn returned, with its input.
type inputIn[EC Context] struct {
	in    *In[EC]
	input *Input[EC]
	// pointer is set for the In of a pointer to the input, and nullable when a null
	// unmarshals to nil.
	pointer, nullable bool
}

// link completes the In from the Go type of the input, as InOf does from the type
// parameter.
func (x inputIn[EC]) link() {
	in, res := x.input, x.in
	t := in.Type
	if !x.pointer {
		res.unmarshal = func(ctx context.Context, ec EC, v any) (any, error) {
			it, replacement, err := in.newValue(ctx, ec, v, false)
			if replacement != nil {
				return replacement, nil
			}
			return reflect.ValueOf(it).Elem().Interface(), graphql.ErrorOnPath(ctx, err)
		}
		res.accept = func(v any) bool { return reflect.TypeOf(v) == t }
		res.zero = reflect.Zero(t).Interface()
		res.name = typeString(t)
		res.nilable = nilable(t)
		return
	}
	pt := reflect.PointerTo(t)
	zero := reflect.Zero(pt).Interface()
	nullable := x.nullable
	res.unmarshal = func(ctx context.Context, ec EC, v any) (any, error) {
		if nullable && v == nil {
			return zero, nil
		}
		// The pointer points to a copy of the value, which an INPUT_OBJECT directive may
		// have replaced, as in the functions mode, so that it is not the pointer that the
		// resolvers that set fields of the input received.
		it, replacement, err := in.newValue(ctx, ec, v, false)
		value := reflect.ValueOf(it).Elem()
		if replacement != nil {
			value = reflect.ValueOf(replacement)
		}
		p := reflect.New(t)
		p.Elem().Set(value)
		return p.Interface(), graphql.ErrorOnPath(ctx, err)
	}
	res.accept = func(v any) bool { return reflect.TypeOf(v) == pt }
	res.zero = zero
	res.name = typeString(pt)
	res.nilable = true
}

// nilable reports whether a value of t can be nil, as the generator's IsNilable does.
func nilable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Interface, reflect.Slice, reflect.Chan:
		return true
	}
	return false
}

// Input describes how a GraphQL input object is unmarshaled into its Go type.
//
// Generated code builds one Input per input object when it builds its tables, with
// Init; the fields refer to the unmarshalers of the tables, which may refer back to the
// input. What the schema says about the input, such as the defaults and directives of
// its fields, Schema.Link reads from the schema. The Go value is created by InputValue
// or InputPointer, or from Type for the index of input unmarshalers, and passed to the
// fields as a pointer in an any, so that one instantiation of the runtime serves every
// input.
type Input[EC Context] struct {
	Name string
	// GoType is the Go type of the input as the generated package writes it. Errors
	// name it.
	GoType string
	// Type is the Go type of the input. The index of input unmarshalers creates its
	// values from it, so that the generated code does not instantiate a function per
	// input for the index.
	Type reflect.Type
	// Pointer is set for inputs generated with return_pointers_in_unmarshalinput: the
	// index of input unmarshalers unmarshals them to a pointer, as InputPointer does.
	Pointer bool
	// IsMap is set when the Go type is map[string]any. The map is made before the
	// fields are set.
	IsMap bool
	// Set stores v, a value of the Go type of the field at index field of the table or
	// nil for its zero value, in it, a pointer to the input's Go type. Generated code
	// declares one per input, a function that switches on the index, and asserts v to the
	// Go type that the field's In unmarshals, so that the compiler checks that the field
	// has that type.
	Set func(it any, field int, v any)

	// fields are processed in this order, which is the order of the schema definition.
	fields []InputField[EC]
	// directives are the INPUT_OBJECT directives, run after all fields are set. They
	// receive the input object as a map, with the defaults of absent fields applied.
	directives []directive[EC]
}

// InputField describes how one field of an input object is stored in Go. Generated
// code sets its name and what ties it to Go; Schema.Link reads the rest from the schema.
type InputField[EC Context] struct {
	Name string
	// In unmarshals the field.
	In *In[EC]
	// Type is set instead of In when Link derives the In from the type of the field and
	// its Go type, which Type gives as a nil pointer to it.
	Type any
	// SetWith is set for fields that a resolver sets, which the Set of the input does
	// not store.
	SetWith func(ctx context.Context, ec EC, it, v any) error
	// GoType is the Go type of the field that errors name, as the generated package
	// writes it, where the runtime would write it otherwise. See Arg.GoType.
	GoType string

	// index is the position of the field in the table, by which the Set of the input
	// stores it.
	index int
	// defaultValue is the default value from the schema, applied when the field is
	// absent. A nil defaultValue means the field has no default.
	defaultValue any
	// directives are the directives declared on the field. They receive the raw input
	// object.
	directives []directive[EC]
	// nilOK is set when a directive may return nil for the field, which then sets the
	// field to its zero value: when the field's Go type can be nil and no resolver
	// sets it.
	nilOK bool
}

// Init sets the input to def, with fields, and registers it with s. Generated code
// calls it once, when it builds the tables of an executable schema.
//
//go:noinline
func (in *Input[EC]) Init(s *Schema[EC], def Input[EC], fields []InputField[EC]) {
	*in = def
	in.fields = fields
	for i := range fields {
		fields[i].index = i
	}
	if _, ok := s.inputs[def.Name]; ok {
		panic("tableimpl:input " + strconv.Quote(def.Name) + " registered twice")
	}
	s.inputs[def.Name] = in
}

// link reads what the tables leave to the schema: see Schema.Link.
func (in *Input[EC]) link(s *Schema[EC], name string) {
	def := s.definition(name)
	in.directives = s.applied(def.Directives, "")
	for i := range in.fields {
		f := &in.fields[i]
		fd := def.Fields.ForName(f.Name)
		if fd == nil {
			panic(fmt.Sprintf("tableimpl:input %s has no field %s", name, f.Name))
		}
		if fd.DefaultValue != nil {
			f.defaultValue = literal(fd.DefaultValue, "the default of "+name+"."+f.Name)
		}
		// The directives of the field's type, such as a scalar, run around the field before
		// its own, as in the functions mode.
		typ := s.definition(fd.Type.Name())
		f.directives = append(
			s.applied(typ.Directives, ast.LocationInputFieldDefinition),
			s.applied(fd.Directives, "")...)
		if f.In == nil {
			t, ok := goType(f.Type)
			if !ok {
				panic(notGoType("input field "+name+"."+f.Name, f.Type))
			}
			f.In = s.derivedIn(fd.Type, t)
		}
		f.nilOK = f.In.nilable && f.SetWith == nil
	}
}

// InputValue unmarshals the raw input object obj into T, the Go type of in.
func InputValue[EC Context, T any](ctx context.Context, ec EC, in *Input[EC], obj any) (T, error) {
	it, replacement, err := in.newValue(ctx, ec, obj, false)
	if replacement != nil {
		return replacement.(T), nil
	}
	return *it.(*T), err
}

// InputPointer is InputValue for inputs generated with
// return_pointers_in_unmarshalinput. An INPUT_OBJECT directive may replace the result
// with another pointer, including a nil one.
func InputPointer[EC Context, T any](
	ctx context.Context,
	ec EC,
	in *Input[EC],
	obj any,
) (*T, error) {
	it, replacement, err := in.newValue(ctx, ec, obj, true)
	if replacement != nil {
		return replacement.(*T), nil
	}
	return it.(*T), err
}

// InputUnmarshalers returns the unmarshalers of the inputs of the tables, bound to ec, for
// the index of input unmarshalers of a request.
func (s *Schema[EC]) InputUnmarshalers(ec EC) []graphql.InputUnmarshaler {
	res := make([]graphql.InputUnmarshaler, 0, len(s.inputs))
	for _, in := range s.inputs {
		goType := in.Type
		if in.Pointer {
			goType = reflect.PointerTo(in.Type)
		}
		res = append(res, graphql.NewUntypedInputUnmarshaler(
			in.Name,
			goType,
			func(ctx context.Context, obj any) (any, error) {
				it, replacement, err := in.newValue(ctx, ec, obj, in.Pointer)
				if replacement != nil {
					return replacement, nil
				}
				if in.Pointer {
					return it, err
				}
				return reflect.ValueOf(it).Elem().Interface(), err
			},
		))
	}
	return res
}

// newValue unmarshals the raw input object obj into a new value of Type, and returns a
// pointer to the value as it. When an INPUT_OBJECT directive replaces the value, it
// returns the replacement, of the Go type of the input, or of a pointer to it when
// pointer is set. A replacement is never nil, but may hold a nil pointer.
//
// It is not generic, so that the unmarshalers of the inputs of a schema share it, and
// the generic functions that call it only convert its results.
func (in *Input[EC]) newValue(
	ctx context.Context,
	ec EC,
	obj any,
	pointer bool,
) (it, replacement any, err error) {
	p := reflect.New(in.Type)
	it = p.Interface()
	want := in.Type
	if pointer {
		want = p.Type()
	}
	// current goes to the directives, through calls that make it escape, so it is only
	// made for an input that has directives.
	var current func() any
	if len(in.directives) > 0 {
		current = func() any { return p.Elem().Interface() }
		if pointer {
			current = p.Interface
		}
	}
	res, replaced, err := in.unmarshal(ctx, ec, obj, it, current)
	if err != nil || !replaced {
		return it, nil, err
	}
	if reflect.TypeOf(res) == want {
		return it, res, nil
	}
	should := in.goType(in.Type)
	if pointer {
		should = "*" + should
	}
	return it, nil, graphql.ErrorOnPath(ctx, fmt.Errorf(
		"unexpected type %T from INPUT_OBJECT directive, should be %s",
		res,
		should,
	))
}

// unmarshal sets the fields of it, a pointer to the Go type of in, from the raw input
// object obj. When in has INPUT_OBJECT directives it returns their result, which
// replaces the input, and replaced is true; the directives receive current().
func (in *Input[EC]) unmarshal(
	ctx context.Context,
	ec EC,
	obj, it any,
	current func() any,
) (res any, replaced bool, err error) {
	if obj == nil {
		return nil, false, nil
	}

	src := obj.(map[string]any)
	// The directives receive the raw object with the defaults filled in. It goes to them
	// through calls that make it escape, so it is only made for an input that has them;
	// otherwise each field reads its raw value or default directly.
	var asMap map[string]any
	if len(in.directives) > 0 {
		asMap = make(map[string]any, len(src))
		maps.Copy(asMap, src)
		for i := range in.fields {
			f := &in.fields[i]
			if f.defaultValue == nil {
				continue
			}
			if _, present := asMap[f.Name]; !present {
				asMap[f.Name] = cloneValue(f.defaultValue)
			}
		}
	}

	if in.IsMap {
		*it.(*map[string]any) = make(map[string]any, len(src))
	}

	for i := range in.fields {
		f := &in.fields[i]
		var v any
		var ok bool
		if asMap != nil {
			v, ok = asMap[f.Name]
		} else if v, ok = src[f.Name]; !ok && f.defaultValue != nil {
			v, ok = cloneValue(f.defaultValue), true
		}
		if !ok {
			continue
		}
		if err := in.unmarshalField(ctx, ec, f, obj, it, v); err != nil {
			return nil, false, err
		}
	}

	if len(in.directives) == 0 {
		return nil, false, nil
	}

	next := func(ctx context.Context) (any, error) { return current(), nil }
	// A directive that cannot run returns the input as it is, as in the functions mode.
	failed := failure{zero: current(), onPath: true}
	res, err = chain(ec, asMap, in.directives, next, failed)(ctx)
	if err != nil {
		return nil, false, graphql.ErrorOnPath(ctx, err)
	}
	return res, true, nil
}

// unmarshalField unmarshals the raw value of the field f into it.
func (in *Input[EC]) unmarshalField(
	ctx context.Context,
	ec EC,
	f *InputField[EC],
	obj, it, raw any,
) error {
	ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField(f.Name))
	if len(f.directives) == 0 {
		data, err := f.In.unmarshal(ctx, ec, raw)
		if err != nil {
			return err
		}
		return in.store(ctx, ec, f, it, data)
	}

	next := func(ctx context.Context) (any, error) { return f.In.unmarshal(ctx, ec, raw) }
	tmp, err := chain(ec, obj, f.directives, next, failure{zero: f.In.zero})(ctx)
	if err != nil {
		return graphql.ErrorOnPath(ctx, err)
	}
	if f.In.accept(tmp) || (tmp == nil && f.nilOK) {
		return in.store(ctx, ec, f, it, tmp)
	}
	goType := f.In.name
	if f.GoType != "" {
		goType = f.GoType
	}
	return graphql.ErrorOnPath(ctx, fmt.Errorf(
		"unexpected type %T from directive, should be %s", tmp, goType))
}

// store stores v, the value of the field f, in it.
func (in *Input[EC]) store(ctx context.Context, ec EC, f *InputField[EC], it, v any) error {
	if f.SetWith != nil {
		return f.SetWith(ctx, ec, it, v)
	}
	// In an input bound to a map, the nil that a directive returns is stored as it is,
	// as the functions mode does, not as a nil of the field's Go type.
	if m, ok := it.(*map[string]any); ok && v == nil {
		(*m)[f.Name] = nil
		return nil
	}
	in.Set(it, f.index, v)
	return nil
}

// goType returns the name of the input's Go type t for errors.
func (in *Input[EC]) goType(t reflect.Type) string {
	if in.GoType != "" {
		return in.GoType
	}
	return t.String()
}

// typeString returns t as go/types writes it, with the full import path of named types,
// which is how the functions mode names types in errors.
func typeString(t reflect.Type) string {
	if t.Name() != "" {
		if t.PkgPath() == "" {
			return t.Name()
		}
		return t.PkgPath() + "." + t.Name()
	}
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + typeString(t.Elem())
	case reflect.Slice:
		return "[]" + typeString(t.Elem())
	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), typeString(t.Elem()))
	case reflect.Map:
		return "map[" + typeString(t.Key()) + "]" + typeString(t.Elem())
	case reflect.Interface:
		if t.NumMethod() == 0 {
			return "any"
		}
	}
	return t.String()
}

// cloneValue returns a copy of v, a value written in the schema, for one request. The
// unmarshalers of scalars such as Map pass a map or a slice on as it is, so a resolver
// that changes the copy would otherwise change the value for every later request.
func cloneValue(v any) any {
	switch v := v.(type) {
	case map[string]any:
		c := make(map[string]any, len(v))
		for k, e := range v {
			c[k] = cloneValue(e)
		}
		return c
	case []any:
		c := make([]any, len(v))
		for i, e := range v {
			c[i] = cloneValue(e)
		}
		return c
	}
	return v
}

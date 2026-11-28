package tableimpl

import (
	"context"
	"reflect"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

// The functions in this file build the marshal and unmarshal functions of the most
// common type references, so that table mode generates a variable holding one of them
// instead of a function. Each one does exactly what the functions mode generates for
// the same type reference.

// The typ parameter of a combinator is the GraphQL type of the reference it marshals,
// such as "Character!". It decides whether a null is an error, and the error names it.

// isNonNull reports whether typ is a non-null type.
func isNonNull(typ string) bool { return strings.HasSuffix(typ, "!") }

// null is the value of a null element. In a non-null position it also reports the
// error the functions mode reports, unless the field already has one.
func null(ctx context.Context, nonNull bool, typ string) graphql.Marshaler {
	if nonNull {
		graphql.AddInvalidNullError(ctx, typ)
	}
	return graphql.Null
}

// nullFromMarshaler is null for a marshaler that returned null for a non-nil value.
func nullFromMarshaler(ctx context.Context, nonNull bool, typ string) graphql.Marshaler {
	if nonNull {
		graphql.AddInvalidNullFromMarshaler(ctx, typ)
	}
	return graphql.Null
}

// MarshalInterface marshals a value of an interface type with m, and nil as null.
//
//go:noinline
func MarshalInterface[EC, V any](typ string, m Marshal[EC, V]) Marshal[EC, V] {
	nonNull := isNonNull(typ)
	return func(ctx context.Context, ec EC, sel ast.SelectionSet, v V) graphql.Marshaler {
		if any(v) == nil {
			return null(ctx, nonNull, typ)
		}
		return m(ctx, ec, sel, v)
	}
}

// MarshalSelf marshals a value that implements graphql.Marshaler.
//
//go:noinline
func MarshalSelf[EC any, V graphql.Marshaler]() Marshal[EC, V] {
	return func(_ context.Context, _ EC, _ ast.SelectionSet, v V) graphql.Marshaler {
		return v
	}
}

// MarshalSelfPtr marshals a pointer that implements graphql.Marshaler, and nil as null.
// The method is looked up at run time rather than required by a constraint, so that
// the elements of one shape, such as the named strings of a schema, share an
// instantiation. Generated code uses it, as MarshalSelf, where it needs the typed
// marshaler; the Out of a field of the type is SelfPtrOut.
//
//go:noinline
func MarshalSelfPtr[EC, E any](typ string) Marshal[EC, *E] {
	nonNull := isNonNull(typ)
	return func(ctx context.Context, _ EC, _ ast.SelectionSet, v *E) graphql.Marshaler {
		if v == nil {
			return null(ctx, nonNull, typ)
		}
		return any(v).(graphql.Marshaler)
	}
}

// MarshalFunc marshals a value with a scalar marshaler such as graphql.MarshalString.
//
//go:noinline
func MarshalFunc[EC, V any](typ string, f func(V) graphql.Marshaler) Marshal[EC, V] {
	nonNull := isNonNull(typ)
	return func(ctx context.Context, _ EC, _ ast.SelectionSet, v V) graphql.Marshaler {
		res := f(v)
		if res == graphql.Null {
			return nullFromMarshaler(ctx, nonNull, typ)
		}
		return res
	}
}

// MarshalFuncPtr marshals a pointer with a scalar marshaler of its element, and nil as
// null.
//
//go:noinline
func MarshalFuncPtr[EC, E any](typ string, f func(E) graphql.Marshaler) Marshal[EC, *E] {
	nonNull := isNonNull(typ)
	return func(ctx context.Context, _ EC, _ ast.SelectionSet, v *E) graphql.Marshaler {
		if v == nil {
			return null(ctx, nonNull, typ)
		}
		res := f(*v)
		if res == graphql.Null {
			return nullFromMarshaler(ctx, nonNull, typ)
		}
		return res
	}
}

// ListOptions describes a list type reference for MarshalList.
type ListOptions struct {
	// Nullable lists marshal nil as null.
	Nullable bool
	// ElemNonNull lists are null when one of their elements is.
	ElemNonNull bool
	// Leaf lists hold scalars, which are marshaled in order without goroutines.
	Leaf bool
	// WorkerLimit and OmitPanicHandler are passed to graphql.MarshalSliceConcurrently.
	WorkerLimit      int64
	OmitPanicHandler bool
}

// MarshalList marshals a slice whose elements are marshaled by elem. Generated code uses
// it where it needs the typed marshaler of a list; the Out of a field of the type is
// ListOut.
//
//go:noinline
func MarshalList[EC, E any](elem Marshal[EC, E], o ListOptions) Marshal[EC, []E] {
	return func(ctx context.Context, ec EC, sel ast.SelectionSet, v []E) graphql.Marshaler {
		return marshalList(ctx, ec, sel, v, elem, o)
	}
}

// marshalList marshals the slice v, whose elements elem marshals, as the functions mode
// does for a list type reference described by o.
func marshalList[EC, E any](
	ctx context.Context,
	ec EC,
	sel ast.SelectionSet,
	v []E,
	elem Marshal[EC, E],
	o ListOptions,
) graphql.Marshaler {
	if o.Nullable && v == nil {
		return graphql.Null
	}
	var ret graphql.Array
	if o.Leaf {
		ret = make(graphql.Array, len(v))
		for i := range v {
			ret[i] = elem(ctx, ec, sel, v[i])
		}
	} else {
		ret = graphql.MarshalSliceConcurrently(ctx, len(v), o.WorkerLimit, o.OmitPanicHandler,
			func(ctx context.Context, i int) graphql.Marshaler {
				fc := graphql.GetFieldContext(ctx)
				fc.Result = &v[i]
				return elem(ctx, ec, sel, v[i])
			})
	}
	if o.ElemNonNull {
		for _, e := range ret {
			if e == graphql.Null {
				return graphql.Null
			}
		}
	}
	return ret
}

// ListOut returns the Out of the slices whose elements elem marshals. It is
// OutOf(MarshalList(elem, o)) in one instantiation, so that a list type reference costs
// one wrapper and one dictionary to compile rather than two of each.
//
//go:noinline
func ListOut[EC, E any](elem Marshal[EC, E], o ListOptions) *Out[EC] {
	return &Out[EC]{
		accept: func(v any) bool {
			_, ok := v.([]E)
			return ok
		},
		marshal: func(ctx context.Context, ec EC, sel ast.SelectionSet, v any) graphql.Marshaler {
			x, _ := v.([]E)
			return marshalList(ctx, ec, sel, x, elem, o)
		},
		name: reflect.TypeFor[[]E]().String(),
		zero: []E(nil),
	}
}

// ListOutOf is ListOut for elements marshaled by an Out rather than a typed marshaler,
// such as the pointers to objects: each element is passed to the Out as any, which costs
// nothing for a pointer. E comes first so that generated code can name it and leave EC to
// be inferred.
//
//go:noinline
func ListOutOf[E, EC any](elem *Out[EC], o ListOptions) *Out[EC] {
	return ListOut(func(ctx context.Context, ec EC, sel ast.SelectionSet, v E) graphql.Marshaler {
		return elem.marshal(ctx, ec, sel, v)
	}, o)
}

// UnmarshalGQL unmarshals a value whose pointer implements graphql.Unmarshaler. The
// method is looked up at run time, as in MarshalSelfPtr.
//
//go:noinline
func UnmarshalGQL[EC, V any]() Unmarshal[EC, V] {
	return func(ctx context.Context, _ EC, v any) (V, error) {
		var res V
		err := any(&res).(graphql.Unmarshaler).UnmarshalGQL(v)
		return res, graphql.ErrorOnPath(ctx, err)
	}
}

// UnmarshalGQLPtr unmarshals a pointer to a value that implements graphql.Unmarshaler.
// Nullable references unmarshal null as nil. The method is looked up at run time, as in
// MarshalSelfPtr.
//
//go:noinline
func UnmarshalGQLPtr[EC, E any](nullable bool) Unmarshal[EC, *E] {
	return func(ctx context.Context, _ EC, v any) (*E, error) {
		if nullable && v == nil {
			return nil, nil
		}
		res := new(E)
		err := any(res).(graphql.Unmarshaler).UnmarshalGQL(v)
		return res, graphql.ErrorOnPath(ctx, err)
	}
}

// UnmarshalFunc unmarshals a value with a scalar unmarshaler such as
// graphql.UnmarshalString.
//
//go:noinline
func UnmarshalFunc[EC, V any](f func(any) (V, error)) Unmarshal[EC, V] {
	return func(ctx context.Context, _ EC, v any) (V, error) {
		res, err := f(v)
		return res, graphql.ErrorOnPath(ctx, err)
	}
}

// UnmarshalFuncPtr unmarshals a pointer with a scalar unmarshaler of its element.
// Nullable references unmarshal null as nil.
//
//go:noinline
func UnmarshalFuncPtr[EC, E any](nullable bool, f func(any) (E, error)) Unmarshal[EC, *E] {
	return func(ctx context.Context, _ EC, v any) (*E, error) {
		if nullable && v == nil {
			return nil, nil
		}
		res, err := f(v)
		return &res, graphql.ErrorOnPath(ctx, err)
	}
}

// UnmarshalInput unmarshals an input object whose Go type is T.
//
//go:noinline
func UnmarshalInput[EC Context, T any](in *Input[EC]) Unmarshal[EC, T] {
	return func(ctx context.Context, ec EC, v any) (T, error) {
		it, replacement, err := in.newValue(ctx, ec, v, false)
		if replacement != nil {
			return replacement.(T), nil
		}
		return *it.(*T), graphql.ErrorOnPath(ctx, err)
	}
}

// UnmarshalInputPtr unmarshals a pointer to an input object, P, a pointer to the Go type
// of in. Nullable references unmarshal null as nil. P is the pointer, not the type it
// points to, so that the inputs of a schema share one instantiation. Generated code uses
// it, as UnmarshalInput, where it needs the typed unmarshaler; the In of an argument or
// input field of the type is InputPtrIn.
//
//go:noinline
func UnmarshalInputPtr[EC Context, P any](nullable bool, in *Input[EC]) Unmarshal[EC, P] {
	return func(ctx context.Context, ec EC, v any) (P, error) {
		if nullable && v == nil {
			var zero P
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
		p := reflect.New(in.Type)
		p.Elem().Set(value)
		return p.Interface().(P), graphql.ErrorOnPath(ctx, err)
	}
}

// MarshalObject marshals a value of an object type, whose Object takes a pointer to it.
//
//go:noinline
func MarshalObject[EC Context, E any](o *Object[EC]) Marshal[EC, E] {
	return func(ctx context.Context, ec EC, sel ast.SelectionSet, v E) graphql.Marshaler {
		return o.Marshal(ctx, ec, sel, &v)
	}
}

// MarshalObjectPtr marshals P, a pointer to a value of an object type, and nil as null.
// P is the pointer, not the type it points to, and its constraint is any, so that the
// objects of a schema share one instantiation: the compiler shares one for pointers only
// under a constraint that has no type set, such as comparable has. A nil P is the zero
// P in an interface. Generated code uses it where it needs the typed marshaler; the Out
// of a field of the type is ObjectPtrOut.
//
//go:noinline
func MarshalObjectPtr[EC Context, P any](typ string, o *Object[EC]) Marshal[EC, P] {
	nonNull := isNonNull(typ)
	var zero P
	return func(ctx context.Context, ec EC, sel ast.SelectionSet, v P) graphql.Marshaler {
		if any(v) == any(zero) {
			return null(ctx, nonNull, typ)
		}
		return o.Marshal(ctx, ec, sel, v)
	}
}

// UnmarshalList unmarshals a list whose elements are unmarshaled by elem. A single
// value is coerced to a list of one. Nullable references unmarshal null as nil. Generated
// code uses it where it needs the typed unmarshaler of a list; the In of an argument or
// input field of the type is ListIn.
//
//go:noinline
func UnmarshalList[EC, E any](nullable bool, elem Unmarshal[EC, E]) Unmarshal[EC, []E] {
	return func(ctx context.Context, ec EC, v any) ([]E, error) {
		return unmarshalList(ctx, ec, v, nullable, elem)
	}
}

// unmarshalList unmarshals the raw list v, whose elements elem unmarshals, as the
// functions mode does.
func unmarshalList[EC, E any](
	ctx context.Context,
	ec EC,
	v any,
	nullable bool,
	elem Unmarshal[EC, E],
) ([]E, error) {
	if nullable && v == nil {
		return nil, nil
	}
	vSlice := graphql.CoerceList(v)
	res := make([]E, len(vSlice))
	for i := range vSlice {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
		var err error
		if res[i], err = elem(ctx, ec, vSlice[i]); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// ListIn returns the In of the slices whose elements elem unmarshals. It is
// InOf(UnmarshalList(nullable, elem)) in one instantiation, as ListOut is for OutOf.
//
//go:noinline
func ListIn[EC, E any](nullable bool, elem Unmarshal[EC, E]) *In[EC] {
	return &In[EC]{
		unmarshal: func(ctx context.Context, ec EC, v any) (any, error) {
			return unmarshalList(ctx, ec, v, nullable, elem)
		},
		accept: func(v any) bool {
			_, ok := v.([]E)
			return ok
		},
		zero:    []E(nil),
		name:    typeString(reflect.TypeFor[[]E]()),
		nilable: true,
	}
}

// ListInOf is ListIn for elements unmarshaled by an In rather than a typed unmarshaler,
// such as the pointers to input objects.
//
//go:noinline
func ListInOf[E, EC any](nullable bool, elem *In[EC]) *In[EC] {
	return ListIn(nullable, func(ctx context.Context, ec EC, v any) (E, error) {
		res, err := elem.unmarshal(ctx, ec, v)
		x, _ := res.(E)
		return x, err
	})
}

// MarshalFuncContext marshals a value with a scalar marshaler that needs the context,
// such as graphql.MarshalFloatContext.
//
//go:noinline
func MarshalFuncContext[EC, V any](
	typ string,
	f func(V) graphql.ContextMarshaler,
) Marshal[EC, V] {
	nonNull := isNonNull(typ)
	return func(ctx context.Context, _ EC, _ ast.SelectionSet, v V) graphql.Marshaler {
		res := f(v)
		if res == graphql.Null {
			nullFromMarshaler(ctx, nonNull, typ)
		}
		return graphql.WrapContextMarshaler(ctx, res)
	}
}

// MarshalFuncContextPtr is MarshalFuncContext for a pointer, and marshals nil as null.
//
//go:noinline
func MarshalFuncContextPtr[EC, E any](
	typ string,
	f func(E) graphql.ContextMarshaler,
) Marshal[EC, *E] {
	nonNull := isNonNull(typ)
	return func(ctx context.Context, _ EC, _ ast.SelectionSet, v *E) graphql.Marshaler {
		if v == nil {
			return null(ctx, nonNull, typ)
		}
		res := f(*v)
		if res == graphql.Null {
			nullFromMarshaler(ctx, nonNull, typ)
		}
		return graphql.WrapContextMarshaler(ctx, res)
	}
}

// UnmarshalFuncContext unmarshals a value with a scalar unmarshaler that needs the
// context, such as graphql.UnmarshalFloatContext.
//
//go:noinline
func UnmarshalFuncContext[EC, V any](f func(context.Context, any) (V, error)) Unmarshal[EC, V] {
	return func(ctx context.Context, _ EC, v any) (V, error) {
		res, err := f(ctx, v)
		return res, graphql.ErrorOnPath(ctx, err)
	}
}

// UnmarshalFuncContextPtr is UnmarshalFuncContext for a pointer. Nullable references
// unmarshal null as nil.
//
//go:noinline
func UnmarshalFuncContextPtr[EC, E any](
	nullable bool,
	f func(context.Context, any) (E, error),
) Unmarshal[EC, *E] {
	return func(ctx context.Context, _ EC, v any) (*E, error) {
		if nullable && v == nil {
			return nil, nil
		}
		res, err := f(ctx, v)
		return &res, graphql.ErrorOnPath(ctx, err)
	}
}

// The constructors below build an Out or an In without a type parameter for the Go type
// of the values, from a nil pointer to the type, zero, which the generated code writes
// as (*T)(nil). The compiler instantiates a generic function once per type argument, and
// a schema has a Go type per object and enum, so the tables of a large schema are much
// smaller to compile when those types share the constructors. The values are checked and
// marshaled through their reflect.Type and the interfaces they implement instead of
// through the type parameter. Link builds the same Outs and Ins for the fields,
// arguments and input fields whose tables give their Go type instead, so generated code
// calls these only for those that a list of its tables refers to; see derive.go.

var (
	marshalerType   = reflect.TypeFor[graphql.Marshaler]()
	unmarshalerType = reflect.TypeFor[graphql.Unmarshaler]()
)

// mustImplement panics when t does not implement iface, which means that the type the
// code was generated for has changed.
func mustImplement(t, iface reflect.Type) {
	if !t.Implements(iface) {
		panic("tableimpl:" + t.String() + " does not implement " + iface.String())
	}
}

// ObjectPtrOut returns the Out of the pointers to the values of o, which marshals nil as
// null. zero is the nil pointer. It is MarshalObjectPtr for the Out of a field.
//
//go:noinline
func ObjectPtrOut[EC Context](typ string, o *Object[EC], zero any) *Out[EC] {
	return objectPtrOut(typ, o, reflect.TypeOf(zero))
}

// objectPtrOut is ObjectPtrOut for the pointer type t.
func objectPtrOut[EC Context](typ string, o *Object[EC], t reflect.Type) *Out[EC] {
	nonNull := isNonNull(typ)
	zero := reflect.Zero(t).Interface()
	return &Out[EC]{
		accept: func(v any) bool { return reflect.TypeOf(v) == t },
		marshal: func(ctx context.Context, ec EC, sel ast.SelectionSet, v any) graphql.Marshaler {
			if v == nil || v == zero {
				return null(ctx, nonNull, typ)
			}
			return o.Marshal(ctx, ec, sel, v)
		},
		name: t.String(),
		zero: zero,
	}
}

// SelfOut returns the Out of the values of a type that implements graphql.Marshaler, such
// as an enum, or a slice or map type with the method, whose nil marshals as null. zero is
// a nil pointer to the type. It is MarshalSelf for the Out of a field.
//
//go:noinline
func SelfOut[EC any](typ string, zero any) *Out[EC] {
	return selfOut[EC](typ, reflect.TypeOf(zero).Elem())
}

// selfOut is SelfOut for the type t.
func selfOut[EC any](typ string, t reflect.Type) *Out[EC] {
	nonNull := isNonNull(typ)
	mustImplement(t, marshalerType)
	marshal := func(_ context.Context, _ EC, _ ast.SelectionSet, v any) graphql.Marshaler {
		return v.(graphql.Marshaler)
	}
	if nilable(t) {
		marshal = func(ctx context.Context, _ EC, _ ast.SelectionSet, v any) graphql.Marshaler {
			if v == nil || reflect.ValueOf(v).IsNil() {
				return null(ctx, nonNull, typ)
			}
			return v.(graphql.Marshaler)
		}
	}
	return &Out[EC]{
		accept:  func(v any) bool { return reflect.TypeOf(v) == t },
		marshal: marshal,
		name:    t.String(),
		zero:    reflect.Zero(t).Interface(),
	}
}

// SelfPtrOut returns the Out of the pointers to the values of a type that implements
// graphql.Marshaler, which marshals nil as null. zero is the nil pointer. It is
// MarshalSelfPtr for the Out of a field.
//
//go:noinline
func SelfPtrOut[EC any](typ string, zero any) *Out[EC] {
	return selfPtrOut[EC](typ, reflect.TypeOf(zero))
}

// selfPtrOut is SelfPtrOut for the pointer type t.
func selfPtrOut[EC any](typ string, t reflect.Type) *Out[EC] {
	nonNull := isNonNull(typ)
	mustImplement(t, marshalerType)
	zero := reflect.Zero(t).Interface()
	return &Out[EC]{
		accept: func(v any) bool { return reflect.TypeOf(v) == t },
		marshal: func(ctx context.Context, _ EC, _ ast.SelectionSet, v any) graphql.Marshaler {
			if v == nil || v == zero {
				return null(ctx, nonNull, typ)
			}
			return v.(graphql.Marshaler)
		},
		name: t.String(),
		zero: zero,
	}
}

// MarshalListOf is MarshalList for elements marshaled by an Out rather than a typed
// marshaler, such as the pointers to objects: each element is passed to the Out as any,
// which costs nothing for a pointer. E comes first so that generated code can name it and
// leave EC to be inferred.
//
//go:noinline
func MarshalListOf[E, EC any](elem *Out[EC], o ListOptions) Marshal[EC, []E] {
	marshal := func(ctx context.Context, ec EC, sel ast.SelectionSet, v E) graphql.Marshaler {
		return elem.marshal(ctx, ec, sel, v)
	}
	return MarshalList(marshal, o)
}

// UnmarshalListOf is UnmarshalList for elements unmarshaled by an In rather than a typed
// unmarshaler, such as the pointers to input objects.
//
//go:noinline
func UnmarshalListOf[E, EC any](nullable bool, elem *In[EC]) Unmarshal[EC, []E] {
	return UnmarshalList(nullable, func(ctx context.Context, ec EC, v any) (E, error) {
		res, err := elem.unmarshal(ctx, ec, v)
		x, _ := res.(E)
		return x, err
	})
}

// GQLIn returns the In of the values of a type whose pointer implements
// graphql.Unmarshaler, such as an enum, or a slice or map type with the method. Nullable
// references of a type that can be nil unmarshal null as nil. zero is a nil pointer to the
// type. It is UnmarshalGQL for the In of an argument or input field.
//
//go:noinline
func GQLIn[EC any](nullable bool, zero any) *In[EC] {
	return gqlIn[EC](nullable, reflect.TypeOf(zero).Elem())
}

// gqlIn is GQLIn for the type t.
func gqlIn[EC any](nullable bool, t reflect.Type) *In[EC] {
	mustImplement(reflect.PointerTo(t), unmarshalerType)
	zeroValue := reflect.Zero(t).Interface()
	nullable = nullable && nilable(t)
	return &In[EC]{
		unmarshal: func(ctx context.Context, _ EC, v any) (any, error) {
			if nullable && v == nil {
				return zeroValue, nil
			}
			p := reflect.New(t)
			err := p.Interface().(graphql.Unmarshaler).UnmarshalGQL(v)
			return p.Elem().Interface(), graphql.ErrorOnPath(ctx, err)
		},
		accept:  func(v any) bool { return reflect.TypeOf(v) == t },
		zero:    zeroValue,
		name:    typeString(t),
		nilable: nilable(t),
	}
}

// GQLPtrIn returns the In of the pointers to the values of a type that implements
// graphql.Unmarshaler. Nullable references unmarshal null as nil. zero is the nil
// pointer. It is UnmarshalGQLPtr for the In of an argument or input field.
//
//go:noinline
func GQLPtrIn[EC any](nullable bool, zero any) *In[EC] {
	return gqlPtrIn[EC](nullable, reflect.TypeOf(zero))
}

// gqlPtrIn is GQLPtrIn for the pointer type pt.
func gqlPtrIn[EC any](nullable bool, pt reflect.Type) *In[EC] {
	mustImplement(pt, unmarshalerType)
	zero := reflect.Zero(pt).Interface()
	return &In[EC]{
		unmarshal: func(ctx context.Context, _ EC, v any) (any, error) {
			if nullable && v == nil {
				return zero, nil
			}
			p := reflect.New(pt.Elem())
			err := p.Interface().(graphql.Unmarshaler).UnmarshalGQL(v)
			return p.Interface(), graphql.ErrorOnPath(ctx, err)
		},
		accept:  func(v any) bool { return reflect.TypeOf(v) == pt },
		zero:    zero,
		name:    typeString(pt),
		nilable: true,
	}
}

// mustString panics when the underlying type of t is not string, which means that the
// type the code was generated for has changed.
func mustString(t reflect.Type) {
	if t.Kind() != reflect.String {
		panic("tableimpl:the underlying type of " + t.String() + " is not string")
	}
}

// CastOut returns the Out of a named type whose underlying type is string and that is
// marshaled by the marshaler of string, f, such as graphql.MarshalString. zero is a nil
// pointer to the type. The functions mode converts the value to string in a function
// generated per type; CastOut reads it through reflect instead, which allocates nothing.
//
//go:noinline
func CastOut[EC any](typ string, f func(string) graphql.Marshaler, zero any) *Out[EC] {
	nonNull := isNonNull(typ)
	t := reflect.TypeOf(zero).Elem()
	mustString(t)
	return &Out[EC]{
		accept: func(v any) bool { return reflect.TypeOf(v) == t },
		marshal: func(ctx context.Context, _ EC, _ ast.SelectionSet, v any) graphql.Marshaler {
			res := f(reflect.ValueOf(v).String())
			if res == graphql.Null {
				return nullFromMarshaler(ctx, nonNull, typ)
			}
			return res
		},
		name: t.String(),
		zero: reflect.Zero(t).Interface(),
	}
}

// CastPtrOut is CastOut for a pointer to such a type, which marshals nil as null. zero is
// the nil pointer.
//
//go:noinline
func CastPtrOut[EC any](typ string, f func(string) graphql.Marshaler, zero any) *Out[EC] {
	nonNull := isNonNull(typ)
	t := reflect.TypeOf(zero)
	mustString(t.Elem())
	return &Out[EC]{
		accept: func(v any) bool { return reflect.TypeOf(v) == t },
		marshal: func(ctx context.Context, _ EC, _ ast.SelectionSet, v any) graphql.Marshaler {
			if v == nil || v == zero {
				return null(ctx, nonNull, typ)
			}
			res := f(reflect.ValueOf(v).Elem().String())
			if res == graphql.Null {
				return nullFromMarshaler(ctx, nonNull, typ)
			}
			return res
		},
		name: t.String(),
		zero: zero,
	}
}

// CastIn returns the In of a named type whose underlying type is string and that is
// unmarshaled by the unmarshaler of string, f, such as graphql.UnmarshalString. zero is a
// nil pointer to the type.
//
//go:noinline
func CastIn[EC any](f func(any) (string, error), zero any) *In[EC] {
	t := reflect.TypeOf(zero).Elem()
	mustString(t)
	return &In[EC]{
		unmarshal: func(ctx context.Context, _ EC, v any) (any, error) {
			s, err := f(v)
			res := reflect.New(t).Elem()
			res.SetString(s)
			return res.Interface(), graphql.ErrorOnPath(ctx, err)
		},
		accept:  func(v any) bool { return reflect.TypeOf(v) == t },
		zero:    reflect.Zero(t).Interface(),
		name:    typeString(t),
		nilable: false,
	}
}

// CastPtrIn is CastIn for a pointer to such a type. Nullable references unmarshal null
// as nil. zero is the nil pointer.
//
//go:noinline
func CastPtrIn[EC any](nullable bool, f func(any) (string, error), zero any) *In[EC] {
	t := reflect.TypeOf(zero)
	mustString(t.Elem())
	return &In[EC]{
		unmarshal: func(ctx context.Context, _ EC, v any) (any, error) {
			if nullable && v == nil {
				return zero, nil
			}
			s, err := f(v)
			res := reflect.New(t.Elem())
			res.Elem().SetString(s)
			return res.Interface(), graphql.ErrorOnPath(ctx, err)
		},
		accept:  func(v any) bool { return reflect.TypeOf(v) == t },
		zero:    zero,
		name:    typeString(t),
		nilable: true,
	}
}

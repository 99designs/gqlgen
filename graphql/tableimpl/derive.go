package tableimpl

import (
	"context"
	"fmt"
	"reflect"

	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

// The tables leave the Out of a field, and the In of an argument or input field, to Link
// when the GraphQL type and the Go type of the values decide it: the generated code then
// gives the Go type as a nil pointer to it, (*V)(nil), instead of building the Out or In.
// Link builds what the generated code would have built for the type: the Out of the
// pointers to an object, of a type that marshals itself or of a list of those, and the
// In of an input object, of a type that unmarshals itself or of a list of those. The
// tables of a large schema have thousands of such type references, which then cost
// nothing to compile.

// typeKey identifies an Out or In that Link derives: the GraphQL type it is for and the
// Go type of its values. Link derives them for named types and lists of named types,
// which the key describes without formatting the GraphQL type, as the tables of a large
// schema look up thousands of them when the schema is created, and for the lists of lists
// bound to a type that marshals itself, whose key holds the formatted type.
type typeKey struct {
	name                    string
	nonNull, list, elemNull bool
	goType                  reflect.Type
}

func keyOf(typ *ast.Type, t reflect.Type) typeKey {
	if typ.Elem == nil {
		return typeKey{name: typ.NamedType, nonNull: typ.NonNull, goType: t}
	}
	// A list of lists has no name, but the Out of a type that marshals itself bound to it
	// writes the GraphQL type in its errors: the key holds the whole type, which no name
	// can be.
	if typ.Elem.Elem != nil {
		return typeKey{name: typ.String(), goType: t}
	}
	return typeKey{
		name:     typ.Elem.NamedType,
		nonNull:  typ.NonNull,
		list:     true,
		elemNull: !typ.Elem.NonNull,
		goType:   t,
	}
}

// goType returns the Go type that zero, a nil pointer to it, gives, and false when zero
// is not such a pointer.
func goType(zero any) (reflect.Type, bool) {
	t := reflect.TypeOf(zero)
	if t == nil || t.Kind() != reflect.Pointer {
		return nil, false
	}
	return t.Elem(), true
}

// notGoType is the message of the panic for zero, the type that of gives in the tables,
// when it is not a nil pointer to a Go type.
func notGoType(of string, zero any) string {
	return fmt.Sprintf("tableimpl:the type of %s is %T, not a nil pointer to a Go type", of, zero)
}

// derivedOut returns the Out of the values of t for a field of the GraphQL type typ.
func (s *Schema[EC]) derivedOut(typ *ast.Type, t reflect.Type) *Out[EC] {
	key := keyOf(typ, t)
	if o := s.outs[key]; o != nil {
		return o
	}
	o := s.newOut(typ, t)
	s.outs[key] = o
	return o
}

func (s *Schema[EC]) newOut(typ *ast.Type, t reflect.Type) *Out[EC] {
	name := typ.String()
	if typ.Elem != nil {
		// A type that marshals itself, such as a named slice type with the method, is
		// bound to the list as a whole, as the functions mode marshals it.
		if t.Implements(marshalerType) {
			if t.Kind() == reflect.Pointer {
				return selfPtrOut[EC](name, t)
			}
			return selfOut[EC](name, t)
		}
		if typ.Elem.Elem != nil || t.Kind() != reflect.Slice {
			panic("tableimpl:no Out to derive for " + t.String() + " as " + name)
		}
		return s.listOut(typ, t)
	}
	def := s.definition(typ.NamedType)
	switch def.Kind {
	case ast.Object:
		o := s.objects[def.Name]
		if o == nil {
			panic("tableimpl:the tables have no object " + def.Name)
		}
		if t.Kind() == reflect.Pointer {
			return objectPtrOut(name, o, t)
		}
	case ast.Scalar, ast.Enum:
		if t.Kind() == reflect.Pointer {
			return selfPtrOut[EC](name, t)
		}
		return selfOut[EC](name, t)
	}
	panic("tableimpl:no Out to derive for " + t.String() + " as " + name)
}

// listOut returns the Out of the slices of type t for the list type typ, whose elements
// are marshaled by their derived Out. It marshals as marshalList does, through reflect.
func (s *Schema[EC]) listOut(typ *ast.Type, t reflect.Type) *Out[EC] {
	elem := s.derivedOut(typ.Elem, t.Elem())
	o := ListOptions{
		Nullable:    !typ.NonNull,
		ElemNonNull: typ.Elem.NonNull,
		// Only scalars count as leaves, as in the generator, not enums.
		Leaf:             s.definition(typ.Elem.Name()).Kind == ast.Scalar,
		WorkerLimit:      s.WorkerLimit,
		OmitPanicHandler: s.OmitPanicHandler,
	}
	return &Out[EC]{
		accept: func(v any) bool { return reflect.TypeOf(v) == t },
		marshal: func(ctx context.Context, ec EC, sel ast.SelectionSet, v any) graphql.Marshaler {
			// A value of another type, or a nil any, is a nil slice, as for ListOut.
			rv := reflect.ValueOf(v)
			if !rv.IsValid() || rv.Type() != t {
				rv = reflect.Zero(t)
			}
			if o.Nullable && rv.IsNil() {
				return graphql.Null
			}
			var ret graphql.Array
			if o.Leaf {
				ret = make(graphql.Array, rv.Len())
				for i := range ret {
					ret[i] = elem.marshal(ctx, ec, sel, rv.Index(i).Interface())
				}
			} else {
				marshal := func(ctx context.Context, i int) graphql.Marshaler {
					e := rv.Index(i)
					fc := graphql.GetFieldContext(ctx)
					fc.Result = e.Addr().Interface()
					return elem.marshal(ctx, ec, sel, e.Interface())
				}
				ret = graphql.MarshalSliceConcurrently(
					ctx, rv.Len(), o.WorkerLimit, o.OmitPanicHandler, marshal)
			}
			if o.ElemNonNull {
				for _, e := range ret {
					if e == graphql.Null {
						return graphql.Null
					}
				}
			}
			return ret
		},
		name: t.String(),
		zero: reflect.Zero(t).Interface(),
	}
}

// derivedIn returns the In of the values of t for an argument or input field of the
// GraphQL type typ.
func (s *Schema[EC]) derivedIn(typ *ast.Type, t reflect.Type) *In[EC] {
	key := keyOf(typ, t)
	if in := s.ins[key]; in != nil {
		return in
	}
	in := s.newIn(typ, t)
	s.ins[key] = in
	return in
}

func (s *Schema[EC]) newIn(typ *ast.Type, t reflect.Type) *In[EC] {
	name := typ.String()
	// A nullable reference unmarshals null as nil, if the Go type can be nil.
	nullable := !typ.NonNull
	if typ.Elem != nil {
		// A type that unmarshals itself is bound to the list as a whole, as for newOut.
		if t.Kind() == reflect.Pointer && t.Implements(unmarshalerType) {
			return gqlPtrIn[EC](nullable, t)
		}
		if reflect.PointerTo(t).Implements(unmarshalerType) {
			return gqlIn[EC](nullable, t)
		}
		if typ.Elem.Elem != nil || t.Kind() != reflect.Slice {
			panic("tableimpl:no In to derive for " + t.String() + " as " + name)
		}
		return s.listIn(typ, t)
	}
	def := s.definition(typ.NamedType)
	switch def.Kind {
	case ast.InputObject:
		in := s.inputs[def.Name]
		if in == nil {
			panic("tableimpl:the tables have no input " + def.Name)
		}
		// The inputs that unmarshal to a pointer or a map keep their unmarshaler.
		if in.Pointer || in.IsMap {
			break
		}
		x := inputIn[EC]{in: &In[EC]{}, input: in}
		if t == in.Type {
			x.link()
			return x.in
		}
		if t == reflect.PointerTo(in.Type) {
			x.pointer, x.nullable = true, nullable
			x.link()
			return x.in
		}
	case ast.Scalar, ast.Enum:
		if t.Kind() == reflect.Pointer {
			return gqlPtrIn[EC](nullable, t)
		}
		return gqlIn[EC](nullable, t)
	}
	panic("tableimpl:no In to derive for " + t.String() + " as " + name)
}

// listIn returns the In of the slices of type t for the list type typ, whose elements
// are unmarshaled by their derived In. It unmarshals as unmarshalList does, through
// reflect.
func (s *Schema[EC]) listIn(typ *ast.Type, t reflect.Type) *In[EC] {
	elem := s.derivedIn(typ.Elem, t.Elem())
	nullable := !typ.NonNull
	elemType := t.Elem()
	zero := reflect.Zero(t).Interface()
	return &In[EC]{
		unmarshal: func(ctx context.Context, ec EC, v any) (any, error) {
			if nullable && v == nil {
				return zero, nil
			}
			vSlice := graphql.CoerceList(v)
			res := reflect.MakeSlice(t, len(vSlice), len(vSlice))
			for i := range vSlice {
				ctx := graphql.WithPathContext(ctx, graphql.NewPathWithIndex(i))
				x, err := elem.unmarshal(ctx, ec, vSlice[i])
				if err != nil {
					return zero, err
				}
				// An element of another type leaves the zero value, as the assertion of
				// ListInOf does.
				if xv := reflect.ValueOf(x); xv.IsValid() && xv.Type() == elemType {
					res.Index(i).Set(xv)
				}
			}
			return res.Interface(), nil
		},
		accept:  func(v any) bool { return reflect.TypeOf(v) == t },
		zero:    zero,
		name:    typeString(t),
		nilable: true,
	}
}

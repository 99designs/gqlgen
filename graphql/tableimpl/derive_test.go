package tableimpl

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

const deriveSchema = `
scalar Shout
type Dog { id: Int }
input Point { x: Int, y: Int, note: String }
type Query { dog: Dog }
`

type dog struct{}

// deriveTestSchema returns a schema with the tables of Dog and Point registered, which
// the derived Outs and Ins of their types use.
func deriveTestSchema() *Schema[*testEC] {
	s := NewSchema[*testEC](gqlparser.MustLoadSchema(&ast.Source{Input: deriveSchema}))
	var o Object[*testEC]
	o.Init(s, "Dog", nil, nil)
	s.inputs["Point"] = pointInput()
	return s
}

// typeOf returns the GraphQL type written as typ, such as "[Dog!]".
func typeOf(typ string) *ast.Type {
	res := &ast.Type{NamedType: strings.TrimSuffix(typ, "!")}
	if inner, ok := strings.CutPrefix(res.NamedType, "["); ok {
		res = &ast.Type{Elem: typeOf(strings.TrimSuffix(inner, "]"))}
	}
	res.NonNull = strings.HasSuffix(typ, "!")
	return res
}

// Link derives the Out of a type reference from its GraphQL type and its Go type, as the
// generated code would build it, once per pair.
func TestDerivedOut(t *testing.T) {
	s := deriveTestSchema()
	ctx := responseCtx()
	v := shout("hey")

	out := s.derivedOut(typeOf("Shout!"), reflect.TypeFor[*shout]())
	require.Same(t, out, s.derivedOut(typeOf("Shout!"), reflect.TypeFor[*shout]()))
	require.NotSame(t, out, s.derivedOut(typeOf("Shout"), reflect.TypeFor[*shout]()))
	require.True(t, out.accept(&v))
	require.Equal(t, `"hey"`, render(out.marshal(ctx, nil, nil, &v)))
	require.Equal(t, graphql.Null, out.marshal(ctx, nil, nil, (*shout)(nil)))
	require.Len(t, graphql.GetErrors(ctx), 1)

	tagsOut := s.derivedOut(typeOf("Shout"), reflect.TypeFor[tags]())
	require.Equal(t, "1", render(tagsOut.marshal(ctx, nil, nil, tags{"a"})))
	require.Equal(t, graphql.Null, tagsOut.marshal(ctx, nil, nil, tags(nil)))

	dogOut := s.derivedOut(typeOf("Dog"), reflect.TypeFor[*dog]())
	require.True(t, dogOut.accept((*dog)(nil)))
	require.False(t, dogOut.accept(dog{}))
	require.Equal(t, graphql.Null, dogOut.marshal(ctx, nil, nil, (*dog)(nil)))

	require.Panics(t, func() { s.derivedOut(typeOf("Dog"), reflect.TypeFor[dog]()) })
	require.Panics(t, func() { s.derivedOut(typeOf("Point"), reflect.TypeFor[*point]()) })
	require.Panics(t, func() { s.derivedOut(typeOf("[[Shout]]"), reflect.TypeFor[[][]*shout]()) })
	require.Panics(t, func() { s.derivedOut(typeOf("[Dog]"), reflect.TypeFor[*dog]()) })

	// A type that marshals itself is bound to a list as a whole, as in the functions mode.
	ctx = responseCtx()
	listTags := s.derivedOut(typeOf("[Shout!]!"), reflect.TypeFor[tags]())
	require.Equal(t, "2", render(listTags.marshal(ctx, nil, nil, tags{"a", "b"})))
	require.Equal(t, graphql.Null, listTags.marshal(ctx, nil, nil, tags(nil)))
	require.ErrorContains(t, graphql.GetErrors(ctx), "[Shout!]!")
	listShout := s.derivedOut(typeOf("[Shout]"), reflect.TypeFor[*shout]())
	require.Equal(t, `"hey"`, render(listShout.marshal(ctx, nil, nil, &v)))

	// Lists of lists bound to one type that marshals itself keep their GraphQL types,
	// which their errors name, apart.
	ctx = responseCtx()
	matrixA := s.derivedOut(typeOf("[[Shout!]!]!"), reflect.TypeFor[tags]())
	matrixB := s.derivedOut(typeOf("[[Int!]!]!"), reflect.TypeFor[tags]())
	require.NotSame(t, matrixA, matrixB)
	require.Same(t, matrixB, s.derivedOut(typeOf("[[Int!]!]!"), reflect.TypeFor[tags]()))
	require.Equal(t, graphql.Null, matrixB.marshal(ctx, nil, nil, tags(nil)))
	require.ErrorContains(t, graphql.GetErrors(ctx), "[[Int!]!]!")
}

// The Out of a list that Link derives marshals as marshalList does.
func TestDerivedListOut(t *testing.T) {
	s := deriveTestSchema()
	ctx := responseCtx()
	v := shout("hey")

	list := s.derivedOut(typeOf("[Shout]!"), reflect.TypeFor[[]*shout]())
	require.True(t, list.accept([]*shout{}))
	require.False(t, list.accept([]shout{}))
	require.Equal(t, "[]*tableimpl.shout", list.name)
	require.Equal(t, `["hey",null]`, render(list.marshal(ctx, nil, nil, []*shout{&v, nil})))
	require.Equal(t, `[]`, render(list.marshal(ctx, nil, nil, []*shout(nil))))
	require.Equal(t, `[]`, render(list.marshal(ctx, nil, nil, nil)))

	nonNull := s.derivedOut(typeOf("[Shout!]"), reflect.TypeFor[[]*shout]())
	require.Equal(t, graphql.Null, nonNull.marshal(ctx, nil, nil, []*shout(nil)))
	require.Equal(t, graphql.Null, nonNull.marshal(ctx, nil, nil, []*shout{&v, nil}))
	// A null anywhere in a list of non-null elements makes the list null.
	require.Equal(t, graphql.Null, nonNull.marshal(ctx, nil, nil, []*shout{&v, &v, &v, nil}))

	// The elements of a list of objects are marshaled in their own field contexts, whose
	// result is the address of the element.
	dogs := s.derivedOut(typeOf("[Dog]"), reflect.TypeFor[[]*dog]())
	require.Equal(t, `[null,null]`, render(dogs.marshal(ctx, nil, nil, []*dog{nil, nil})))
	nonNullDogs := s.derivedOut(typeOf("[Dog!]!"), reflect.TypeFor[[]*dog]())
	require.Equal(t, graphql.Null, nonNullDogs.marshal(ctx, nil, nil, []*dog{nil}))
}

// Link derives the In of a type reference as it derives its Out.
func TestDerivedIn(t *testing.T) {
	s := deriveTestSchema()
	ctx := context.Background()

	in := s.derivedIn(typeOf("Shout"), reflect.TypeFor[*shout]())
	require.Same(t, in, s.derivedIn(typeOf("Shout"), reflect.TypeFor[*shout]()))
	v, err := in.unmarshal(ctx, nil, nil)
	require.NoError(t, err)
	require.Equal(t, (*shout)(nil), v)
	v, err = in.unmarshal(ctx, nil, "hi")
	require.NoError(t, err)
	require.Equal(t, shout("hi!"), *v.(*shout))

	// A non-null reference unmarshals null through the method.
	_, err = s.derivedIn(typeOf("Shout!"), reflect.TypeFor[*shout]()).unmarshal(ctx, nil, nil)
	require.EqualError(t, err, "input: not a string")
	v, err = s.derivedIn(typeOf("Shout!"), reflect.TypeFor[shout]()).unmarshal(ctx, nil, "a")
	require.NoError(t, err)
	require.Equal(t, shout("a!"), v)

	p, err := s.derivedIn(typeOf("Point!"), reflect.TypeFor[point]()).
		unmarshal(ctx, nil, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, point{X: 1, Y: 7}, p)
	ptr := s.derivedIn(typeOf("Point"), reflect.TypeFor[*point]())
	p, err = ptr.unmarshal(ctx, nil, nil)
	require.NoError(t, err)
	require.Equal(t, (*point)(nil), p)
	p, err = ptr.unmarshal(ctx, nil, map[string]any{"x": 2})
	require.NoError(t, err)
	require.Equal(t, &point{X: 2, Y: 7}, p)

	require.Panics(t, func() { s.derivedIn(typeOf("Point"), reflect.TypeFor[**point]()) })
	require.Panics(t, func() { s.derivedIn(typeOf("Dog"), reflect.TypeFor[*dog]()) })

	// A type that unmarshals itself is bound to a list as a whole, as for the Out.
	listTags := s.derivedIn(typeOf("[Shout!]"), reflect.TypeFor[tags]())
	v, err = listTags.unmarshal(ctx, nil, nil)
	require.NoError(t, err)
	require.Equal(t, tags(nil), v)
	v, err = listTags.unmarshal(ctx, nil, "a")
	require.NoError(t, err)
	require.Equal(t, tags{"a"}, v)
	v, err = s.derivedIn(typeOf("[Shout]"), reflect.TypeFor[*shout]()).unmarshal(ctx, nil, "hi")
	require.NoError(t, err)
	require.Equal(t, shout("hi!"), *v.(*shout))
}

// The In of a list that Link derives unmarshals as unmarshalList does.
func TestDerivedListIn(t *testing.T) {
	s := deriveTestSchema()
	ctx := context.Background()

	list := s.derivedIn(typeOf("[Shout!]"), reflect.TypeFor[[]shout]())
	require.True(t, list.accept([]shout{}))
	require.False(t, list.accept([]*shout{}))
	require.Equal(t, []shout(nil), list.zero)
	require.True(t, list.nilable)
	require.Equal(t, "[]github.com/99designs/gqlgen/graphql/tableimpl.shout", list.name)
	v, err := list.unmarshal(ctx, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []shout(nil), v)
	v, err = list.unmarshal(ctx, nil, "a")
	require.NoError(t, err)
	require.Equal(t, []shout{"a!"}, v)
	// Every element is unmarshaled, however long the list.
	v, err = list.unmarshal(ctx, nil, []any{"a", "b", "c", "d"})
	require.NoError(t, err)
	require.Equal(t, []shout{"a!", "b!", "c!", "d!"}, v)
	v, err = list.unmarshal(ctx, nil, []any{"a", 1})
	require.Error(t, err)
	require.Equal(t, []shout(nil), v)

	// A non-null list unmarshals null as an empty list.
	v, err = s.derivedIn(typeOf("[Shout!]!"), reflect.TypeFor[[]shout]()).unmarshal(ctx, nil, nil)
	require.NoError(t, err)
	require.Equal(t, []shout{}, v)

	points := s.derivedIn(typeOf("[Point]"), reflect.TypeFor[[]*point]())
	v, err = points.unmarshal(ctx, nil, []any{map[string]any{"x": 3}, nil})
	require.NoError(t, err)
	require.Equal(t, []*point{{X: 3, Y: 7}, nil}, v)
}

// Link derives the Outs and Ins that the tables give as Go types.
func TestLinkDerives(t *testing.T) {
	s := NewSchema[*testEC](gqlparser.MustLoadSchema(&ast.Source{Input: `
scalar Shout
type Query { echo(s: Shout, n: Int): Shout }
input In { s: Shout }
`}))
	var query Object[*testEC]
	query.Init(s, "Query", nil, []Field[*testEC]{
		{
			Name: "echo",
			Args: []Arg[*testEC]{{Name: "s", Type: (**shout)(nil)}, {Name: "n"}},
			Type: (**shout)(nil),
		},
	})
	var in Input[*testEC]
	in.Init(s, Input[*testEC]{
		Name: "In",
		Type: reflect.TypeFor[struct{ S *shout }](),
		Set:  func(any, int, any) {},
	}, []InputField[*testEC]{{Name: "s", Type: (**shout)(nil)}})
	s.Link()

	f := query.index["echo"]
	require.True(t, f.Out.accept((*shout)(nil)))
	require.Equal(t, "*tableimpl.shout", f.Out.name)
	require.Len(t, f.args, 1)
	require.True(t, f.args[0].typ.accept((*shout)(nil)))
	require.True(t, f.args[0].nilOK)
	// The argument and the input field share the In of their type.
	require.Same(t, f.args[0].typ, in.fields[0].In)
	require.True(t, in.fields[0].nilOK)

	// A Type must be a nil pointer to the Go type.
	s = NewSchema[*testEC](gqlparser.MustLoadSchema(&ast.Source{Input: `
scalar Shout
type Query { echo: Shout }
`}))
	query.Init(s, "Query", nil, []Field[*testEC]{{Name: "echo", Type: shout("")}})
	require.PanicsWithValue(t,
		"tableimpl:the type of field echo is tableimpl.shout, not a nil pointer to a Go type",
		s.Link)
}

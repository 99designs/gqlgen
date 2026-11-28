package tableimpl

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

func render(m graphql.Marshaler) string {
	var buf bytes.Buffer
	m.MarshalGQL(&buf)
	return buf.String()
}

func responseCtx() context.Context {
	ctx := graphql.WithResponseContext(context.Background(), graphql.DefaultErrorPresenter, nil)
	return graphql.WithFieldContext(ctx, &graphql.FieldContext{})
}

func TestMarshalNull(t *testing.T) {
	ctx := responseCtx()
	m := MarshalFuncPtr[*testEC]("String", graphql.MarshalString)
	require.Equal(t, graphql.Null, m(ctx, nil, nil, nil))
	require.Empty(t, graphql.GetErrors(ctx))

	m = MarshalFuncPtr[*testEC]("String!", graphql.MarshalString)
	require.Equal(t, graphql.Null, m(ctx, nil, nil, nil))
	require.Len(t, graphql.GetErrors(ctx), 1)
	require.ErrorIs(t, graphql.GetErrors(ctx)[0], graphql.ErrInvalidNull)
	require.Equal(
		t,
		"cannot return null for non-null position (String!): the value was nil",
		graphql.GetErrors(ctx)[0].Message,
	)

	s := "x"
	require.Equal(t, `"x"`, render(m(ctx, nil, nil, &s)))

	// A marshaler that returns null for a value is reported as such.
	ctx = responseCtx()
	mt := MarshalFunc[*testEC]("Time!", graphql.MarshalTime)
	require.Equal(t, graphql.Null, mt(ctx, nil, nil, time.Time{}))
	require.Len(t, graphql.GetErrors(ctx), 1)
	require.Equal(
		t,
		"cannot return null for non-null position (Time!): "+
			"the marshaler returned null for a non-nil value",
		graphql.GetErrors(ctx)[0].Message,
	)
}

func TestMarshalList(t *testing.T) {
	ctx := responseCtx()
	elem := MarshalFuncPtr[*testEC]("String!", graphql.MarshalString)
	s := "a"

	require.Equal(
		t,
		`["a"]`,
		render(MarshalList(elem, ListOptions{Leaf: true})(ctx, nil, nil, []*string{&s})),
	)
	require.Equal(
		t,
		graphql.Null,
		MarshalList(elem, ListOptions{Nullable: true})(ctx, nil, nil, nil),
	)
	require.Equal(t, `[]`, render(MarshalList(elem, ListOptions{})(ctx, nil, nil, nil)))
	// A null element of a list of non-null elements makes the list null.
	require.Equal(
		t,
		graphql.Null,
		MarshalList(
			elem,
			ListOptions{ElemNonNull: true, Leaf: true},
		)(
			ctx,
			nil,
			nil,
			[]*string{&s, nil},
		),
	)
	require.Equal(
		t,
		`["a",null]`,
		render(
			MarshalList(
				MarshalFuncPtr[*testEC]("String", graphql.MarshalString),
				ListOptions{Leaf: true},
			)(
				ctx,
				nil,
				nil,
				[]*string{&s, nil},
			),
		),
	)
}

// ListOut and ListIn are OutOf(MarshalList) and InOf(UnmarshalList) in one; ListOutOf
// and ListInOf take the Out or In of the elements instead of a typed combinator.
func TestListCombinators(t *testing.T) {
	ctx := responseCtx()
	s := "a"
	out := ListOut(
		MarshalFuncPtr[*testEC]("String!", graphql.MarshalString),
		ListOptions{Leaf: true, ElemNonNull: true},
	)
	require.True(t, out.accept([]*string{}))
	require.False(t, out.accept([]string{}))
	require.Equal(t, "[]*string", out.name)
	require.Equal(t, `["a"]`, render(out.marshal(ctx, nil, nil, []*string{&s})))
	require.Equal(t, graphql.Null, out.marshal(ctx, nil, nil, []*string{&s, nil}))
	// A null anywhere in a list of non-null elements makes the list null.
	require.Equal(t, graphql.Null, out.marshal(ctx, nil, nil, []*string{&s, &s, &s, nil}))
	require.Equal(t, `[]`, render(out.marshal(ctx, nil, nil, nil)))
	nullable := ListOut(
		MarshalFuncPtr[*testEC]("String!", graphql.MarshalString),
		ListOptions{Nullable: true, Leaf: true},
	)
	require.Equal(t, graphql.Null, nullable.marshal(ctx, nil, nil, nil))

	v := shout("hey")
	ofOut := ListOutOf[*shout](SelfPtrOut[*testEC]("Shout", (*shout)(nil)), ListOptions{Leaf: true})
	require.True(t, ofOut.accept([]*shout{}))
	require.Equal(t, `["hey",null]`, render(ofOut.marshal(ctx, nil, nil, []*shout{&v, nil})))

	in := ListIn(true, unmarshalInt)
	require.True(t, in.accept([]int{}))
	require.False(t, in.accept([]string{}))
	require.Equal(t, "[]int", in.name)
	require.Equal(t, []int(nil), in.zero)
	require.True(t, in.nilable)
	list, err := in.unmarshal(ctx, nil, 3)
	require.NoError(t, err)
	require.Equal(t, []int{3}, list)
	// Every element is unmarshaled, however long the list.
	list, err = in.unmarshal(ctx, nil, []any{1, 2, 3, 4, 5})
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3, 4, 5}, list)
	list, err = in.unmarshal(ctx, nil, nil)
	require.NoError(t, err)
	require.Nil(t, list)
	_, err = ListIn(false, unmarshalInt).unmarshal(ctx, nil, []any{1, "two"})
	require.EqualError(t, err, "not an int")

	ofIn := ListInOf[*shout](false, GQLPtrIn[*testEC](true, (*shout)(nil)))
	require.True(t, ofIn.accept([]*shout{}))
	got, err := ofIn.unmarshal(ctx, nil, []any{"hi", nil})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, shout("hi!"), *got.([]*shout)[0])
	require.Nil(t, got.([]*shout)[1])
}

func TestUnmarshalCombinators(t *testing.T) {
	ctx := context.Background()

	list, err := UnmarshalList(true, unmarshalInt)(ctx, nil, 3)
	require.NoError(t, err)
	require.Equal(t, []int{3}, list)
	list, err = UnmarshalList(true, unmarshalInt)(ctx, nil, nil)
	require.NoError(t, err)
	require.Nil(t, list)
	_, err = UnmarshalList(false, unmarshalInt)(ctx, nil, []any{1, "two"})
	require.EqualError(t, err, "not an int")

	in := pointInput()
	p, err := UnmarshalInputPtr[*testEC, *point](true, in)(ctx, nil, nil)
	require.NoError(t, err)
	require.Nil(t, p)
	p, err = UnmarshalInputPtr[*testEC, *point](false, in)(ctx, nil, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, &point{X: 1, Y: 7}, p)
	// The pointer points to the value an INPUT_OBJECT directive replaces the input with.
	in.directives = returning(point{X: 9}, nil)
	p, err = UnmarshalInputPtr[*testEC, *point](false, in)(ctx, nil, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, &point{X: 9}, p)

	v, err := UnmarshalInput[*testEC, point](in)(ctx, nil, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, point{X: 9}, v)
	in.directives = nil
	v, err = UnmarshalInput[*testEC, point](in)(ctx, nil, map[string]any{"x": 2})
	require.NoError(t, err)
	require.Equal(t, point{X: 2, Y: 7}, v)

	sp, err := UnmarshalFuncPtr[*testEC](true, graphql.UnmarshalString)(ctx, nil, nil)
	require.NoError(t, err)
	require.Nil(t, sp)
	sp, err = UnmarshalFuncPtr[*testEC](false, graphql.UnmarshalString)(ctx, nil, "s")
	require.NoError(t, err)
	require.Equal(t, "s", *sp)
}

// shout is a scalar that marshals and unmarshals itself, with its methods on the pointer.
type shout string

func (s *shout) UnmarshalGQL(v any) error {
	str, ok := v.(string)
	if !ok {
		return errors.New("not a string")
	}
	*s = shout(str + "!")
	return nil
}

func (s *shout) MarshalGQL(w io.Writer) { _, _ = io.WriteString(w, strconv.Quote(string(*s))) }

// The combinators of types that marshal or unmarshal themselves find the methods at run
// time.
func TestSelfCombinators(t *testing.T) {
	ctx := responseCtx()

	v, err := UnmarshalGQL[*testEC, shout]()(ctx, nil, "hi")
	require.NoError(t, err)
	require.Equal(t, shout("hi!"), v)
	_, err = UnmarshalGQL[*testEC, shout]()(ctx, nil, 1)
	require.EqualError(t, err, "input: not a string")

	p, err := UnmarshalGQLPtr[*testEC, shout](true)(ctx, nil, nil)
	require.NoError(t, err)
	require.Nil(t, p)
	p, err = UnmarshalGQLPtr[*testEC, shout](false)(ctx, nil, "hey")
	require.NoError(t, err)
	require.Equal(t, shout("hey!"), *p)

	m := MarshalSelfPtr[*testEC, shout]("Shout!")
	require.Equal(t, `"hey!"`, render(m(ctx, nil, nil, p)))
	require.Equal(t, graphql.Null, m(ctx, nil, nil, nil))
	require.Len(t, graphql.GetErrors(ctx), 1)
}

// code is a named string bound to a String scalar, which the functions mode marshals by
// converting it to string.
type code string

// The Outs and Ins of casts read and write the string through reflect.
func TestCastCombinators(t *testing.T) {
	ctx := responseCtx()

	out := CastOut[*testEC]("Code!", graphql.MarshalString, (*code)(nil))
	require.True(t, out.accept(code("x")))
	require.False(t, out.accept("x"))
	require.Equal(t, `"x"`, render(out.marshal(ctx, nil, nil, code("x"))))
	require.Equal(t, "tableimpl.code", out.name)

	ptr := CastPtrOut[*testEC]("Code!", graphql.MarshalString, (*code)(nil))
	c := code("y")
	require.True(t, ptr.accept(&c))
	require.False(t, ptr.accept(c))
	require.Equal(t, `"y"`, render(ptr.marshal(ctx, nil, nil, &c)))
	require.Equal(t, graphql.Null, ptr.marshal(ctx, nil, nil, (*code)(nil)))
	require.Len(t, graphql.GetErrors(ctx), 1)
	require.ErrorIs(t, graphql.GetErrors(ctx)[0], graphql.ErrInvalidNull)

	in := CastIn[*testEC](graphql.UnmarshalString, (*code)(nil))
	v, err := in.unmarshal(ctx, nil, "z")
	require.NoError(t, err)
	require.Equal(t, code("z"), v)
	require.Equal(t, code(""), in.zero)
	require.False(t, in.nilable)
	_, err = in.unmarshal(ctx, nil, []any{})
	require.Error(t, err)

	pin := CastPtrIn[*testEC](true, graphql.UnmarshalString, (*code)(nil))
	v, err = pin.unmarshal(ctx, nil, nil)
	require.NoError(t, err)
	require.Equal(t, (*code)(nil), v)
	v, err = pin.unmarshal(ctx, nil, "w")
	require.NoError(t, err)
	require.Equal(t, code("w"), *v.(*code))
	require.True(t, pin.nilable)

	require.Panics(t, func() { CastOut[*testEC]("Code!", graphql.MarshalString, (*int)(nil)) })
}

// tags is a slice type that marshals and unmarshals itself, whose nil is null.
type tags []string

func (t tags) MarshalGQL(w io.Writer) { _, _ = io.WriteString(w, strconv.Itoa(len(t))) }

func (t *tags) UnmarshalGQL(v any) error {
	s, ok := v.(string)
	if !ok {
		return errors.New("not a string")
	}
	*t = tags{s}
	return nil
}

// The Outs and Ins of types that marshal themselves check the nil of a slice or map.
func TestSelfCombinatorsNilable(t *testing.T) {
	ctx := responseCtx()

	out := SelfOut[*testEC]("Tags!", (*tags)(nil))
	require.True(t, out.accept(tags{}))
	require.False(t, out.accept([]string{}))
	require.Equal(t, "2", render(out.marshal(ctx, nil, nil, tags{"a", "b"})))
	require.Equal(t, graphql.Null, out.marshal(ctx, nil, nil, tags(nil)))
	require.Len(t, graphql.GetErrors(ctx), 1)
	require.ErrorIs(t, graphql.GetErrors(ctx)[0], graphql.ErrInvalidNull)
	ctx = responseCtx()
	nullable := SelfOut[*testEC]("Tags", (*tags)(nil))
	require.Equal(t, graphql.Null, nullable.marshal(ctx, nil, nil, tags(nil)))
	require.Empty(t, graphql.GetErrors(ctx))

	in := GQLIn[*testEC](true, (*tags)(nil))
	v, err := in.unmarshal(ctx, nil, nil)
	require.NoError(t, err)
	require.Equal(t, tags(nil), v)
	v, err = in.unmarshal(ctx, nil, "x")
	require.NoError(t, err)
	require.Equal(t, tags{"x"}, v)
	require.True(t, in.nilable)

	// A non-null reference unmarshals null through the method, as the functions mode does.
	_, err = GQLIn[*testEC](false, (*tags)(nil)).unmarshal(ctx, nil, nil)
	require.EqualError(t, err, "input: not a string")

	// A value type ignores nullable: it cannot be nil.
	e, err := GQLIn[*testEC](true, (*shout)(nil)).unmarshal(ctx, nil, "hi")
	require.NoError(t, err)
	require.Equal(t, shout("hi!"), e)
}

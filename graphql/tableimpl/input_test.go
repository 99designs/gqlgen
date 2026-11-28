package tableimpl

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/99designs/gqlgen/graphql"
)

type testEC struct{}

func (*testEC) OpCtx() *graphql.OperationContext { return nil }

func (*testEC) AddDeferred(int32) {}

func (*testEC) ProcessDeferredGroup(graphql.DeferredGroup) {}

type point struct {
	X, Y int
	Note *string
}

func unmarshalInt(_ context.Context, _ *testEC, v any) (int, error) {
	i, ok := v.(int)
	if !ok {
		return 0, errors.New("not an int")
	}
	return i, nil
}

func unmarshalString(_ context.Context, _ *testEC, v any) (*string, error) {
	if v == nil {
		return nil, nil
	}
	s, ok := v.(string)
	if !ok {
		return nil, errors.New("not a string")
	}
	return &s, nil
}

var (
	inInt    = InOf(unmarshalInt)
	inString = InOf(unmarshalString)
)

// indexed numbers fields as Init does, for the inputs that the tests build without it.
func indexed(fields []InputField[*testEC]) []InputField[*testEC] {
	for i := range fields {
		fields[i].index = i
	}
	return fields
}

func pointInput() *Input[*testEC] {
	return &Input[*testEC]{
		Name: "Point",
		Type: reflect.TypeFor[point](),
		Set: func(it any, field int, v any) {
			p := it.(*point)
			switch field {
			case 0:
				p.X, _ = v.(int)
			case 1:
				p.Y, _ = v.(int)
			case 2:
				p.Note, _ = v.(*string)
			}
		},
		fields: indexed([]InputField[*testEC]{
			{Name: "x", In: inInt.In},
			{Name: "y", defaultValue: 7, In: inInt.In},
			{Name: "note", In: inString.In},
		}),
	}
}

func TestInputUnmarshal(t *testing.T) {
	ctx := context.Background()
	in := pointInput()

	p, err := InputValue[*testEC, point](ctx, nil, in, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, point{X: 1, Y: 7}, p)

	p, err = InputValue[*testEC, point](ctx, nil, in, nil)
	require.NoError(t, err)
	require.Equal(t, point{}, p)

	pp, err := InputPointer[*testEC, point](ctx, nil, in, map[string]any{"x": 2, "y": 3})
	require.NoError(t, err)
	require.Equal(t, &point{X: 2, Y: 3}, pp)

	_, err = InputValue[*testEC, point](ctx, nil, in, map[string]any{"x": "one"})
	require.EqualError(t, err, "not an int")
}

// The index of input unmarshalers unmarshals the inputs of the tables as InputValue and
// InputPointer do, into either shape of target.
func TestInputUnmarshalers(t *testing.T) {
	s := NewSchema[*testEC](nil)
	value, pointer := pointInput(), pointInput()
	pointer.Name, pointer.Pointer = "PointPointer", true
	s.inputs[value.Name], s.inputs[pointer.Name] = value, pointer
	ctx := graphql.WithInputUnmarshalerIndex(
		context.Background(),
		graphql.NewInputUnmarshalerIndex(s.InputUnmarshalers(nil)...),
	)
	unmarshal := func(name string, raw, v any) error {
		return graphql.UnmarshalNamedInputFromContext(ctx, name, raw, v)
	}
	x := func(x int) map[string]any { return map[string]any{"x": x} }

	for _, name := range []string{"Point", "PointPointer"} {
		t.Run(name, func(t *testing.T) {
			var p point
			require.NoError(t, unmarshal(name, x(4), &p))
			require.Equal(t, point{X: 4, Y: 7}, p)

			var pp *point
			require.NoError(t, unmarshal(name, nil, &pp))
			require.Equal(t, &point{}, pp)

			require.EqualError(t, unmarshal(name, map[string]any{"x": "one"}, &p), "not an int")
		})
	}

	value.directives = returning(point{X: 9}, nil)
	var p point
	require.NoError(t, unmarshal("Point", x(1), &p))
	require.Equal(t, point{X: 9}, p)

	// With pointers a directive may replace the input with nil.
	pointer.directives = returning((*point)(nil), nil)
	pp := &point{X: 1}
	require.NoError(t, unmarshal("PointPointer", x(1), &pp))
	require.Nil(t, pp)

	const wrong = "input: unexpected type %s from INPUT_OBJECT directive, should be %s"
	value.directives = returning(&point{}, nil)
	require.EqualError(
		t,
		unmarshal("Point", x(1), &p),
		fmt.Sprintf(wrong, "*tableimpl.point", "tableimpl.point"),
	)
	pointer.directives = returning(point{}, nil)
	require.EqualError(
		t,
		unmarshal("PointPointer", x(1), &p),
		fmt.Sprintf(wrong, "tableimpl.point", "*tableimpl.point"),
	)
}

// InputIn unmarshals an input by value as UnmarshalInput does, and Link completes the
// rest of the In from the Go type of the input.
func TestInputIn(t *testing.T) {
	ctx := context.Background()
	s := NewSchema[*testEC](nil)
	in := pointInput()
	res := InputIn(s, in)
	s.Link()

	require.Equal(t, point{}, res.zero)
	require.Equal(t, "github.com/99designs/gqlgen/graphql/tableimpl.point", res.name)
	require.False(t, res.nilable)
	require.True(t, res.accept(point{}))
	require.False(t, res.accept(&point{}))
	require.False(t, res.accept(nil))

	v, err := res.unmarshal(ctx, nil, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, point{X: 1, Y: 7}, v)
	_, err = res.unmarshal(ctx, nil, map[string]any{"x": "one"})
	require.EqualError(t, err, "input: not an int")

	in.directives = returning(point{X: 9}, nil)
	v, err = res.unmarshal(ctx, nil, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, point{X: 9}, v)
}

func TestInputFieldDirectives(t *testing.T) {
	ctx := graphql.WithPathContext(context.Background(), graphql.NewPathWithField("arg"))
	in := pointInput()

	in.fields[2].directives = returning(nil, nil)
	in.fields[2].nilOK = true
	p, err := InputValue[*testEC, point](ctx, nil, in, map[string]any{"note": "dropped"})
	require.NoError(t, err)
	require.Nil(t, p.Note)

	in.fields[0].directives = returning(nil, nil)
	_, err = InputValue[*testEC, point](ctx, nil, in, map[string]any{"x": 1})
	require.EqualError(t, err, "input: arg.x unexpected type <nil> from directive, should be int")

	in.fields[0].directives = returning("one", nil)
	_, err = InputValue[*testEC, point](ctx, nil, in, map[string]any{"x": 1})
	require.EqualError(t, err, "input: arg.x unexpected type string from directive, should be int")
	// The tables may give the Go type that errors name, as the generated package writes it.
	in.fields[0].GoType = "github.com/x/app.Count"
	_, err = InputValue[*testEC, point](ctx, nil, in, map[string]any{"x": 1})
	require.EqualError(t, err,
		"input: arg.x unexpected type string from directive, should be github.com/x/app.Count")
	in.fields[0].GoType = ""

	in.fields[0].directives = returning(5, nil)
	p, err = InputValue[*testEC, point](ctx, nil, in, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, 5, p.X)
}

func TestInputObjectDirectives(t *testing.T) {
	ctx := context.Background()
	in := pointInput()
	replace := func(v any) []directive[*testEC] {
		def := &DirectiveDef[*testEC]{
			Name: "replace",
			Call: func(ctx context.Context, ec *testEC, obj any, next graphql.Resolver, args map[string]any) (any, error) {
				require.IsType(t, map[string]any{}, obj)
				got, err := next(ctx)
				if err != nil || v == nil {
					return got, err
				}
				return v, nil
			},
		}
		return []directive[*testEC]{{def: def}}
	}

	in.directives = replace(nil)
	p, err := InputValue[*testEC, point](ctx, nil, in, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, point{X: 1, Y: 7}, p)
	pp, err := InputPointer[*testEC, point](ctx, nil, in, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, &point{X: 1, Y: 7}, pp)

	in.directives = replace(point{X: 9})
	p, err = InputValue[*testEC, point](ctx, nil, in, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Equal(t, point{X: 9}, p)

	// With pointers a directive may replace the input with nil.
	in.directives = replace((*point)(nil))
	pp, err = InputPointer[*testEC, point](ctx, nil, in, map[string]any{"x": 1})
	require.NoError(t, err)
	require.Nil(t, pp)

	in.directives = replace("wrong")
	_, err = InputValue[*testEC, point](ctx, nil, in, map[string]any{"x": 1})
	require.EqualError(
		t,
		err,
		"input: unexpected type string from INPUT_OBJECT directive, should be tableimpl.point",
	)
}

func TestInputMap(t *testing.T) {
	in := &Input[*testEC]{
		Name:  "Map",
		Type:  reflect.TypeFor[map[string]any](),
		IsMap: true,
		Set:   func(it any, _ int, v any) { x, _ := v.(int); (*it.(*map[string]any))["x"] = x },
		fields: indexed([]InputField[*testEC]{
			{Name: "x", In: inInt.In},
		}),
	}
	m, err := InputValue[*testEC, map[string]any](
		context.Background(),
		nil,
		in,
		map[string]any{"x": 1, "ignored": true},
	)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"x": 1}, m)
}

func TestInputFieldSetByResolver(t *testing.T) {
	in := &Input[*testEC]{
		Name: "Point",
		Type: reflect.TypeFor[point](),
		fields: []InputField[*testEC]{
			{
				Name: "x",
				In:   inInt.In,
				SetWith: func(ctx context.Context, ec *testEC, it, v any) error {
					x, _ := v.(int)
					if x < 0 {
						return errors.New("negative")
					}
					it.(*point).X = x * 10
					return nil
				},
			},
		},
	}
	p, err := InputValue[*testEC, point](context.Background(), nil, in, map[string]any{"x": 2})
	require.NoError(t, err)
	require.Equal(t, 20, p.X)

	_, err = InputValue[*testEC, point](context.Background(), nil, in, map[string]any{"x": -1})
	require.EqualError(t, err, "negative")
}

// returning returns directives that call next and then return v instead of its result.
// onCall, when set, is called with the object the directive receives.
func returning(v any, onCall func(obj any)) []directive[*testEC] {
	def := &DirectiveDef[*testEC]{
		Name: "returning",
		Call: func(ctx context.Context, ec *testEC, obj any, next graphql.Resolver, args map[string]any) (any, error) {
			if onCall != nil {
				onCall(obj)
			}
			if _, err := next(ctx); err != nil {
				return nil, err
			}
			return v, nil
		},
	}
	return []directive[*testEC]{{def: def}}
}

func TestTypeString(t *testing.T) {
	require.Equal(t, "int", typeString(reflect.TypeFor[int]()))
	require.Equal(t, "*string", typeString(reflect.TypeFor[*string]()))
	require.Equal(
		t,
		"[]*github.com/99designs/gqlgen/graphql/tableimpl.point",
		typeString(reflect.TypeFor[[]*point]()),
	)
	require.Equal(t, "map[string]any", typeString(reflect.TypeFor[map[string]any]()))
	require.Equal(
		t,
		"github.com/99designs/gqlgen/graphql.Marshaler",
		typeString(reflect.TypeFor[graphql.Marshaler]()),
	)
}

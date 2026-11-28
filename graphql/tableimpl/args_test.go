package tableimpl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	ctx := context.Background()
	args := []arg[*testEC]{
		{name: "x", typ: inInt.In},
		{name: "note", typ: inString.In},
	}

	got, err := parseArgs(ctx, nil, args, map[string]any{"x": 3}, true)
	require.NoError(t, err)
	// Absent arguments hold the zero value of their Go type.
	require.Equal(t, map[string]any{"x": 3, "note": (*string)(nil)}, got)

	_, err = parseArgs(ctx, nil, args, map[string]any{"x": "three"}, true)
	require.EqualError(t, err, "not an int")
}

func TestParseArgsDirectives(t *testing.T) {
	ctx := context.Background()
	var calls int
	count := func(obj any) {
		calls++
		require.IsType(t, map[string]any{}, obj)
	}
	args := []arg[*testEC]{
		{name: "note", nilOK: true, directives: returning(nil, count), typ: inString.In},
	}
	got, err := parseArgs(ctx, nil, args, map[string]any{"note": "x"}, true)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"note": (*string)(nil)}, got)
	require.Equal(t, 1, calls)

	// Directives are skipped for absent arguments unless withNull is set.
	_, err = parseArgs(ctx, nil, args, map[string]any{}, true)
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	args[0].withNull = true
	_, err = parseArgs(ctx, nil, args, map[string]any{}, true)
	require.NoError(t, err)
	require.Equal(t, 2, calls)

	args = []arg[*testEC]{{name: "x", directives: returning(nil, count), typ: inInt.In}}
	_, err = parseArgs(ctx, nil, args, map[string]any{"x": 1}, true)
	require.EqualError(t, err, "input: x unexpected type <nil> from directive, should be int")
	// The tables may give the Go type that errors name, as the generated package writes it.
	args[0].goType = "github.com/x/app.Count"
	_, err = parseArgs(ctx, nil, args, map[string]any{"x": 1}, true)
	require.EqualError(t, err,
		"input: x unexpected type <nil> from directive, should be github.com/x/app.Count")
	args[0].goType = ""

	args[0].directives = returning(8, count)
	got, err = parseArgs(ctx, nil, args, map[string]any{"x": 1}, true)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"x": 8}, got)
}

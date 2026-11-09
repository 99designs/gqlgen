package graphql_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/99designs/gqlgen/graphql"
)

func TestMarshalEnum(t *testing.T) {
	for _, tc := range []struct {
		name  string
		raw   string
		valid bool
	}{
		{"valid", `"OK"`, true},
		{"escaped valid", `"\u004fK"`, true},
		{"unknown", `"OTHER"`, false},
		{"empty", `""`, false},
		{"null", `null`, false},
		{"number", `1`, false},
		{"malformed", `"OK`, false},
		{"multiple values", `"OK" "FAIL"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := graphql.WithResponseContext(
				t.Context(),
				graphql.DefaultErrorPresenter,
				graphql.DefaultRecover,
			)
			ctx = graphql.WithPathContext(ctx, graphql.NewPathWithField("status"))
			calls := 0
			input := graphql.WriterFunc(func(w io.Writer) {
				calls++
				io.WriteString(w, tc.raw)
			})
			result := graphql.MarshalEnum(ctx, "Status", input, "OK", "FAIL")
			require.Equal(t, 1, calls)
			if !tc.valid {
				require.Same(t, graphql.Null, result)
				errs := graphql.GetErrors(ctx)
				require.Len(t, errs, 1)
				require.Equal(t, ast.Path{ast.PathName("status")}, errs[0].Path)
				require.Equal(t, "invalid value for enum Status", errs[0].Message)
				return
			}
			require.Empty(t, graphql.GetErrors(ctx))
			var buf bytes.Buffer
			result.MarshalGQL(&buf)
			require.Equal(t, tc.raw, buf.String())
			require.Equal(t, 1, calls, "validation must not invoke custom marshalers twice")
		})
	}
}

func TestMarshalEnumContextError(t *testing.T) {
	ctx := graphql.WithResponseContext(
		t.Context(),
		graphql.DefaultErrorPresenter,
		graphql.DefaultRecover,
	)
	input := graphql.WrapContextMarshaler(
		ctx,
		graphql.ContextWriterFunc(func(ctx context.Context, w io.Writer) error {
			io.WriteString(w, `"partial`)
			return errors.New("enum marshal failed")
		}),
	)
	result := graphql.MarshalEnum(ctx, "Status", input, "OK")
	require.Same(t, graphql.Null, result)
	errs := graphql.GetErrors(ctx)
	require.Len(t, errs, 1)
	require.Equal(t, "enum marshal failed", errs[0].Message)
	var buf bytes.Buffer
	result.MarshalGQL(&buf)
	require.Equal(t, "null", buf.String(), "partial output must be discarded")
}

func BenchmarkMarshalEnum(b *testing.B) {
	ctx := graphql.WithResponseContext(
		b.Context(),
		graphql.DefaultErrorPresenter,
		graphql.DefaultRecover,
	)
	value := graphql.MarshalString("OK")
	b.ReportAllocs()
	for b.Loop() {
		graphql.MarshalEnum(ctx, "Status", value, "OK", "FAIL").MarshalGQL(io.Discard)
	}
}

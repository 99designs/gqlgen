package graphql

import (
	"bytes"
	"context"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

type eventCtxKey string

func TestResolveFieldStreamWithEventContext(t *testing.T) {
	t.Run("yields per-event context attached to Event.Context", func(t *testing.T) {
		ch := make(chan Event[string], 2)
		ch <- Event[string]{
			Context: context.WithValue(context.Background(), eventCtxKey("id"), "evt-1"),
			Value:   "first",
		}
		ch <- Event[string]{
			Context: context.WithValue(context.Background(), eventCtxKey("id"), "evt-2"),
			Value:   "second",
		}
		close(ch)

		next := makeWithEventContextResolver(t, (<-chan Event[string])(ch))

		ctx1, m1 := next(context.Background())
		assertEventID(ctx1, t, "evt-1")
		assertMarshalsTo(t, m1, `{"testField":"first"}`)

		ctx2, m2 := next(context.Background())
		assertEventID(ctx2, t, "evt-2")
		assertMarshalsTo(t, m2, `{"testField":"second"}`)
	})

	t.Run("returns nil marshaler when channel is closed", func(t *testing.T) {
		ch := make(chan Event[string])
		close(ch)

		next := makeWithEventContextResolver(t, (<-chan Event[string])(ch))
		ctx, m := next(context.Background())
		if m != nil {
			t.Fatalf("expected nil marshaler on closed channel, got %T", m)
		}
		if ctx == nil {
			t.Fatal("expected non-nil ctx even on closed channel")
		}
	})

	t.Run("falls back to input ctx when Event.Context is nil", func(t *testing.T) {
		ch := make(chan Event[string], 1)
		ch <- Event[string]{Context: nil, Value: "x"}
		close(ch)

		next := makeWithEventContextResolver(t, (<-chan Event[string])(ch))
		input := context.WithValue(context.Background(), eventCtxKey("id"), "input")
		got, m := next(input)
		if m == nil {
			t.Fatal("expected non-nil marshaler")
		}
		assertEventID(got, t, "input")
	})

	t.Run("returns nil when input context is cancelled and channel is empty", func(t *testing.T) {
		ch := make(chan Event[string])
		next := makeWithEventContextResolver(t, (<-chan Event[string])(ch))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		got, m := next(ctx)
		if m != nil {
			t.Fatalf("expected nil marshaler on cancelled ctx, got %T", m)
		}
		if got == nil {
			t.Fatal("expected non-nil ctx even on cancellation")
		}
	})
}

// makeWithEventContextResolver builds a one-shot wrapper around
// ResolveFieldStreamWithEventContext that bypasses the full middleware harness,
// returning the per-iteration function used by the dispatcher.
func makeWithEventContextResolver(
	t *testing.T,
	source <-chan Event[string],
) func(context.Context) (context.Context, Marshaler) {
	t.Helper()
	ctx := WithResponseContext(context.Background(), DefaultErrorPresenter, nil)
	oc := &OperationContext{
		ResolverMiddleware: func(ctx context.Context, next Resolver) (any, error) {
			return next(ctx)
		},
	}
	field := CollectedField{Field: &ast.Field{Alias: "testField"}}

	return ResolveFieldStreamWithEventContext(
		ctx,
		oc,
		field,
		func(_ context.Context, field CollectedField) (*FieldContext, error) {
			return &FieldContext{Object: "Test", Field: field}, nil
		},
		func(_ context.Context) (any, error) {
			return source, nil
		},
		nil,
		func(_ context.Context, _ ast.SelectionSet, v string) Marshaler {
			return MarshalString(v)
		},
		false,
		false,
	)
}

func assertEventID(ctx context.Context, t *testing.T, want string) {
	t.Helper()
	got, _ := ctx.Value(eventCtxKey("id")).(string)
	if got != want {
		t.Fatalf("expected event id %q in ctx, got %q", want, got)
	}
}

func assertMarshalsTo(t *testing.T, m Marshaler, want string) {
	t.Helper()
	var buf bytes.Buffer
	m.MarshalGQL(&buf)
	if buf.String() != want {
		t.Fatalf("marshaled output mismatch:\n got: %s\nwant: %s", buf.String(), want)
	}
}

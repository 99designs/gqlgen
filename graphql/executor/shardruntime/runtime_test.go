package shardruntime

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestFieldRegistry(t *testing.T) {
	resetFieldRegistryForTest()

	h := func(context.Context, ObjectExecutionContext, graphql.CollectedField, any) graphql.Marshaler {
		return graphql.Null
	}

	if got, ok := LookupField("scope", "Query", "name"); ok || got != nil {
		t.Fatalf("unexpected field handler before registration: handler=%v ok=%v", got, ok)
	}

	RegisterField("scope", "Query", "name", h)

	got, ok := LookupField("scope", "Query", "name")
	if !ok {
		t.Fatal("expected registered field handler")
	}
	if got == nil {
		t.Fatal("expected non-nil field handler")
	}

	if got, ok := LookupField("scope", "Query", "missing"); ok || got != nil {
		t.Fatalf("unexpected field handler for missing field: handler=%v ok=%v", got, ok)
	}
	if got, ok := LookupField("scope", "Mutation", "name"); ok || got != nil {
		t.Fatalf("unexpected field handler for missing object: handler=%v ok=%v", got, ok)
	}
	if got, ok := LookupField("other-scope", "Query", "name"); ok || got != nil {
		t.Fatalf("unexpected field handler for missing scope: handler=%v ok=%v", got, ok)
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate registration panic")
		}
		msg, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected panic string, got %T", recovered)
		}
		expected := "duplicate field shard handler registration: scope:Query:name"
		if msg != expected {
			t.Fatalf("unexpected panic message: got %q want %q", msg, expected)
		}
	}()

	RegisterField("scope", "Query", "name", h)
}

func TestFieldLookupSnapshotIsBuiltLazily(t *testing.T) {
	resetFieldRegistryForTest()

	h := func(context.Context, ObjectExecutionContext, graphql.CollectedField, any) graphql.Marshaler {
		return graphql.Null
	}

	const total = 16
	for i := 0; i < total; i++ {
		RegisterField("scope", "Query", fmt.Sprintf("field_%03d", i), h)
	}

	if got := len(loadFieldLookupSnapshot()); got != 0 {
		t.Fatalf("unexpected eager field snapshot size: got %d want 0", got)
	}

	got, ok := LookupField("scope", "Query", "field_000")
	if !ok || got == nil {
		t.Fatal("expected lookup to resolve registered field after lazy snapshot build")
	}

	if got := len(loadFieldLookupSnapshot()); got != total {
		t.Fatalf("unexpected rebuilt field snapshot size: got %d want %d", got, total)
	}
}

func TestStreamFieldRegistry(t *testing.T) {
	resetStreamFieldRegistryForTest()

	h := func(context.Context, ObjectExecutionContext, graphql.CollectedField, any) func(context.Context) graphql.Marshaler {
		return func(context.Context) graphql.Marshaler {
			return graphql.Null
		}
	}

	if got, ok := LookupStreamField("scope", "Query", "name"); ok || got != nil {
		t.Fatalf("unexpected stream field handler before registration: handler=%v ok=%v", got, ok)
	}

	RegisterStreamField("scope", "Query", "name", h)

	got, ok := LookupStreamField("scope", "Query", "name")
	if !ok {
		t.Fatal("expected registered stream field handler")
	}
	if got == nil {
		t.Fatal("expected non-nil stream field handler")
	}

	if got, ok := LookupStreamField("scope", "Query", "missing"); ok || got != nil {
		t.Fatalf("unexpected stream field handler for missing field: handler=%v ok=%v", got, ok)
	}
	if got, ok := LookupStreamField("scope", "Mutation", "name"); ok || got != nil {
		t.Fatalf("unexpected stream field handler for missing object: handler=%v ok=%v", got, ok)
	}
	if got, ok := LookupStreamField("other-scope", "Query", "name"); ok || got != nil {
		t.Fatalf("unexpected stream field handler for missing scope: handler=%v ok=%v", got, ok)
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate registration panic")
		}
		msg, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected panic string, got %T", recovered)
		}
		expected := "duplicate stream field shard handler registration: scope:Query:name"
		if msg != expected {
			t.Fatalf("unexpected panic message: got %q want %q", msg, expected)
		}
	}()

	RegisterStreamField("scope", "Query", "name", h)
}

func TestComplexityRegistry(t *testing.T) {
	resetComplexityRegistryForTest()

	h := func(context.Context, ObjectExecutionContext, int, map[string]any) (int, bool) {
		return 42, true
	}

	if got, ok := LookupComplexity("scope", "Query", "name"); ok || got != nil {
		t.Fatalf("unexpected complexity handler before registration: handler=%v ok=%v", got, ok)
	}

	RegisterComplexity("scope", "Query", "name", h)

	got, ok := LookupComplexity("scope", "Query", "name")
	if !ok {
		t.Fatal("expected registered complexity handler")
	}
	if got == nil {
		t.Fatal("expected non-nil complexity handler")
	}

	if got, ok := LookupComplexity("scope", "Query", "missing"); ok || got != nil {
		t.Fatalf("unexpected complexity handler for missing field: handler=%v ok=%v", got, ok)
	}
	if got, ok := LookupComplexity("scope", "Mutation", "name"); ok || got != nil {
		t.Fatalf("unexpected complexity handler for missing object: handler=%v ok=%v", got, ok)
	}
	if got, ok := LookupComplexity("other-scope", "Query", "name"); ok || got != nil {
		t.Fatalf("unexpected complexity handler for missing scope: handler=%v ok=%v", got, ok)
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate registration panic")
		}
		msg, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected panic string, got %T", recovered)
		}
		expected := "duplicate complexity shard handler registration: scope:Query:name"
		if msg != expected {
			t.Fatalf("unexpected panic message: got %q want %q", msg, expected)
		}
	}()

	RegisterComplexity("scope", "Query", "name", h)
}

func TestInputUnmarshalRegistryDeterministicOrder(t *testing.T) {
	resetInputUnmarshalRegistryForTest()

	type marker struct{ id string }
	inputB := &marker{id: "B"}
	inputA := &marker{id: "A"}
	inputC := &marker{id: "C"}

	if got := ListInputUnmarshalers("scope", nil); got != nil {
		t.Fatalf("unexpected input unmarshalers before registration: %v", got)
	}

	RegisterInputUnmarshaler("scope", "InputB", inputB)
	RegisterInputUnmarshaler("scope", "InputA", inputA)
	RegisterInputUnmarshaler("scope", "InputC", inputC)
	RegisterInputUnmarshaler("other-scope", "InputA", &marker{id: "other"})

	got := ListInputUnmarshalers("scope", nil)
	if len(got) != 3 {
		t.Fatalf("unexpected number of input unmarshalers: got %d want %d", len(got), 3)
	}

	if got[0] != inputA {
		t.Fatalf("unexpected first input unmarshaler: got %v want %v", got[0], inputA)
	}
	if got[1] != inputB {
		t.Fatalf("unexpected second input unmarshaler: got %v want %v", got[1], inputB)
	}
	if got[2] != inputC {
		t.Fatalf("unexpected third input unmarshaler: got %v want %v", got[2], inputC)
	}

	if got := ListInputUnmarshalers("missing-scope", nil); got != nil {
		t.Fatalf("unexpected input unmarshalers for missing scope: %v", got)
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate registration panic")
		}
		msg, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected panic string, got %T", recovered)
		}
		expected := "duplicate input unmarshaler registration: scope:InputA"
		if msg != expected {
			t.Fatalf("unexpected panic message: got %q want %q", msg, expected)
		}
	}()

	RegisterInputUnmarshaler("scope", "InputA", &marker{id: "dup"})
}

func TestInputUnmarshalMap(t *testing.T) {
	resetInputUnmarshalRegistryForTest()

	type inputA struct{ Value string }
	type inputB struct{ Value string }

	fnA := func(context.Context, any) (inputA, error) { return inputA{}, nil }
	fnB := func(context.Context, any) (inputB, error) { return inputB{}, nil }

	RegisterInputUnmarshaler("scope", "InputA", fnA)
	RegisterInputUnmarshaler("scope", "InputB", fnB)

	inputMap := InputUnmarshalMap("scope", nil)
	if len(inputMap) != 2 {
		t.Fatalf("unexpected number of input unmarshalers in map: got %d want %d", len(inputMap), 2)
	}

	if _, ok := inputMap[reflect.TypeFor[inputA]()]; !ok {
		t.Fatal("missing inputA unmarshaler in map")
	}
	if _, ok := inputMap[reflect.TypeFor[inputB]()]; !ok {
		t.Fatal("missing inputB unmarshaler in map")
	}

	missingScope := InputUnmarshalMap("missing-scope", nil)
	if len(missingScope) != 0 {
		t.Fatalf("expected empty input unmarshaler map for missing scope, got %d entries", len(missingScope))
	}
}

func TestCodecMarshalRegistry(t *testing.T) {
	resetCodecMarshalRegistryForTest()

	h := func(context.Context, ObjectExecutionContext, ast.SelectionSet, any) graphql.Marshaler {
		return graphql.Null
	}

	if got, ok := LookupCodecMarshal("scope", "marshalFoo"); ok || got != nil {
		t.Fatalf("unexpected codec marshal handler before registration: handler=%v ok=%v", got, ok)
	}

	RegisterCodecMarshal("scope", "marshalFoo", h)

	got, ok := LookupCodecMarshal("scope", "marshalFoo")
	if !ok {
		t.Fatal("expected registered codec marshal handler")
	}
	if got == nil {
		t.Fatal("expected non-nil codec marshal handler")
	}

	if got, ok := LookupCodecMarshal("scope", "marshalMissing"); ok || got != nil {
		t.Fatalf("unexpected codec marshal handler for missing func: handler=%v ok=%v", got, ok)
	}
	if got, ok := LookupCodecMarshal("other-scope", "marshalFoo"); ok || got != nil {
		t.Fatalf("unexpected codec marshal handler for missing scope: handler=%v ok=%v", got, ok)
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate registration panic")
		}
		msg, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected panic string, got %T", recovered)
		}
		expected := "duplicate codec marshal handler registration: scope:marshalFoo"
		if msg != expected {
			t.Fatalf("unexpected panic message: got %q want %q", msg, expected)
		}
	}()

	RegisterCodecMarshal("scope", "marshalFoo", h)
}

func TestCodecUnmarshalRegistry(t *testing.T) {
	resetCodecUnmarshalRegistryForTest()

	h := func(context.Context, ObjectExecutionContext, any) (any, error) {
		return nil, nil
	}

	if got, ok := LookupCodecUnmarshal("scope", "unmarshalBar"); ok || got != nil {
		t.Fatalf("unexpected codec unmarshal handler before registration: handler=%v ok=%v", got, ok)
	}

	RegisterCodecUnmarshal("scope", "unmarshalBar", h)

	got, ok := LookupCodecUnmarshal("scope", "unmarshalBar")
	if !ok {
		t.Fatal("expected registered codec unmarshal handler")
	}
	if got == nil {
		t.Fatal("expected non-nil codec unmarshal handler")
	}

	if got, ok := LookupCodecUnmarshal("scope", "unmarshalMissing"); ok || got != nil {
		t.Fatalf("unexpected codec unmarshal handler for missing func: handler=%v ok=%v", got, ok)
	}
	if got, ok := LookupCodecUnmarshal("other-scope", "unmarshalBar"); ok || got != nil {
		t.Fatalf("unexpected codec unmarshal handler for missing scope: handler=%v ok=%v", got, ok)
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate registration panic")
		}
		msg, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected panic string, got %T", recovered)
		}
		expected := "duplicate codec unmarshal handler registration: scope:unmarshalBar"
		if msg != expected {
			t.Fatalf("unexpected panic message: got %q want %q", msg, expected)
		}
	}()

	RegisterCodecUnmarshal("scope", "unmarshalBar", h)
}

func TestRegistryDuplicatePanics(t *testing.T) {
	t.Run("object", func(t *testing.T) {
		resetObjectRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, ast.SelectionSet, any) graphql.Marshaler {
			return graphql.Null
		}

		RegisterObject("scope", "Query", func(ctx context.Context, ec ObjectExecutionContext, sel ast.SelectionSet, obj any) graphql.Marshaler {
			return h(ctx, ec, sel, obj)
		})
		assertDuplicateRegistrationPanic(t, "duplicate object shard handler registration: scope:Query", func() {
			RegisterObject("scope", "Query", func(ctx context.Context, ec ObjectExecutionContext, sel ast.SelectionSet, obj any) graphql.Marshaler {
				return h(ctx, ec, sel, obj)
			})
		})
	})

	t.Run("stream object", func(t *testing.T) {
		resetStreamObjectRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, ast.SelectionSet) func(context.Context) graphql.Marshaler {
			return func(context.Context) graphql.Marshaler {
				return graphql.Null
			}
		}

		RegisterStreamObject("scope", "Query", func(ctx context.Context, ec ObjectExecutionContext, sel ast.SelectionSet) func(context.Context) graphql.Marshaler {
			return h(ctx, ec, sel)
		})
		assertDuplicateRegistrationPanic(t, "duplicate stream object shard handler registration: scope:Query", func() {
			RegisterStreamObject("scope", "Query", func(ctx context.Context, ec ObjectExecutionContext, sel ast.SelectionSet) func(context.Context) graphql.Marshaler {
				return h(ctx, ec, sel)
			})
		})
	})

	t.Run("field", func(t *testing.T) {
		resetFieldRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, graphql.CollectedField, any) graphql.Marshaler {
			return graphql.Null
		}

		RegisterField("scope", "Query", "name", h)
		assertDuplicateRegistrationPanic(t, "duplicate field shard handler registration: scope:Query:name", func() {
			RegisterField("scope", "Query", "name", h)
		})
	})

	t.Run("stream field", func(t *testing.T) {
		resetStreamFieldRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, graphql.CollectedField, any) func(context.Context) graphql.Marshaler {
			return func(context.Context) graphql.Marshaler {
				return graphql.Null
			}
		}

		RegisterStreamField("scope", "Query", "name", h)
		assertDuplicateRegistrationPanic(t, "duplicate stream field shard handler registration: scope:Query:name", func() {
			RegisterStreamField("scope", "Query", "name", h)
		})
	})

	t.Run("complexity", func(t *testing.T) {
		resetComplexityRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, int, map[string]any) (int, bool) {
			return 42, true
		}

		RegisterComplexity("scope", "Query", "name", h)
		assertDuplicateRegistrationPanic(t, "duplicate complexity shard handler registration: scope:Query:name", func() {
			RegisterComplexity("scope", "Query", "name", h)
		})
	})

	t.Run("input unmarshaler", func(t *testing.T) {
		resetInputUnmarshalRegistryForTest()

		type marker struct{ id string }

		RegisterInputUnmarshaler("scope", "InputA", &marker{id: "A"})
		assertDuplicateRegistrationPanic(t, "duplicate input unmarshaler registration: scope:InputA", func() {
			RegisterInputUnmarshaler("scope", "InputA", &marker{id: "dup"})
		})
	})

	t.Run("codec marshal", func(t *testing.T) {
		resetCodecMarshalRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, ast.SelectionSet, any) graphql.Marshaler {
			return graphql.Null
		}

		RegisterCodecMarshal("scope", "marshalFoo", h)
		assertDuplicateRegistrationPanic(t, "duplicate codec marshal handler registration: scope:marshalFoo", func() {
			RegisterCodecMarshal("scope", "marshalFoo", h)
		})
	})

	t.Run("codec unmarshal", func(t *testing.T) {
		resetCodecUnmarshalRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, any) (any, error) {
			return nil, nil
		}

		RegisterCodecUnmarshal("scope", "unmarshalBar", h)
		assertDuplicateRegistrationPanic(t, "duplicate codec unmarshal handler registration: scope:unmarshalBar", func() {
			RegisterCodecUnmarshal("scope", "unmarshalBar", h)
		})
	})
}

func TestRegistryConcurrentAccess(t *testing.T) {
	t.Run("field", func(t *testing.T) {
		resetFieldRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, graphql.CollectedField, any) graphql.Marshaler {
			return graphql.Null
		}

		const total = 128
		const readers = 8

		errCh := make(chan error, total+readers)
		reportErr := func(err error) {
			select {
			case errCh <- err:
			default:
			}
		}

		start := make(chan struct{})
		var writersWG sync.WaitGroup
		var readersWG sync.WaitGroup
		var writesDone atomic.Bool

		for i := 0; i < total; i++ {
			writersWG.Add(1)
			go func(i int) {
				defer writersWG.Done()
				<-start
				RegisterField("scope", "Query", fmt.Sprintf("field_%03d", i), h)
			}(i)
		}

		for i := 0; i < readers; i++ {
			readersWG.Add(1)
			go func() {
				defer readersWG.Done()
				<-start
				for !writesDone.Load() {
					for i := 0; i < total; i++ {
						handler, ok := LookupField("scope", "Query", fmt.Sprintf("field_%03d", i))
						if ok && handler == nil {
							reportErr(fmt.Errorf("nil field handler for registered key %d", i))
						}
					}
				}
			}()
		}

		close(start)
		writersWG.Wait()
		writesDone.Store(true)
		readersWG.Wait()

		close(errCh)
		for err := range errCh {
			t.Fatal(err)
		}

		for i := 0; i < total; i++ {
			handler, ok := LookupField("scope", "Query", fmt.Sprintf("field_%03d", i))
			if !ok || handler == nil {
				t.Fatalf("missing registered field handler for key %d", i)
			}
		}
	})

	t.Run("stream field", func(t *testing.T) {
		resetStreamFieldRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, graphql.CollectedField, any) func(context.Context) graphql.Marshaler {
			return func(context.Context) graphql.Marshaler {
				return graphql.Null
			}
		}

		const total = 128
		const readers = 8

		errCh := make(chan error, total+readers)
		reportErr := func(err error) {
			select {
			case errCh <- err:
			default:
			}
		}

		start := make(chan struct{})
		var writersWG sync.WaitGroup
		var readersWG sync.WaitGroup
		var writesDone atomic.Bool

		for i := 0; i < total; i++ {
			writersWG.Add(1)
			go func(i int) {
				defer writersWG.Done()
				<-start
				RegisterStreamField("scope", "Query", fmt.Sprintf("stream_field_%03d", i), h)
			}(i)
		}

		for i := 0; i < readers; i++ {
			readersWG.Add(1)
			go func() {
				defer readersWG.Done()
				<-start
				for !writesDone.Load() {
					for i := 0; i < total; i++ {
						handler, ok := LookupStreamField("scope", "Query", fmt.Sprintf("stream_field_%03d", i))
						if ok && handler == nil {
							reportErr(fmt.Errorf("nil stream field handler for registered key %d", i))
						}
					}
				}
			}()
		}

		close(start)
		writersWG.Wait()
		writesDone.Store(true)
		readersWG.Wait()

		close(errCh)
		for err := range errCh {
			t.Fatal(err)
		}

		for i := 0; i < total; i++ {
			handler, ok := LookupStreamField("scope", "Query", fmt.Sprintf("stream_field_%03d", i))
			if !ok || handler == nil {
				t.Fatalf("missing registered stream field handler for key %d", i)
			}
		}
	})

	t.Run("complexity", func(t *testing.T) {
		resetComplexityRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, int, map[string]any) (int, bool) {
			return 42, true
		}

		const total = 128
		const readers = 8

		errCh := make(chan error, total+readers)
		reportErr := func(err error) {
			select {
			case errCh <- err:
			default:
			}
		}

		start := make(chan struct{})
		var writersWG sync.WaitGroup
		var readersWG sync.WaitGroup
		var writesDone atomic.Bool

		for i := 0; i < total; i++ {
			writersWG.Add(1)
			go func(i int) {
				defer writersWG.Done()
				<-start
				RegisterComplexity("scope", "Query", fmt.Sprintf("complexity_%03d", i), h)
			}(i)
		}

		for i := 0; i < readers; i++ {
			readersWG.Add(1)
			go func() {
				defer readersWG.Done()
				<-start
				for !writesDone.Load() {
					for i := 0; i < total; i++ {
						handler, ok := LookupComplexity("scope", "Query", fmt.Sprintf("complexity_%03d", i))
						if ok && handler == nil {
							reportErr(fmt.Errorf("nil complexity handler for registered key %d", i))
						}
					}
				}
			}()
		}

		close(start)
		writersWG.Wait()
		writesDone.Store(true)
		readersWG.Wait()

		close(errCh)
		for err := range errCh {
			t.Fatal(err)
		}

		for i := 0; i < total; i++ {
			handler, ok := LookupComplexity("scope", "Query", fmt.Sprintf("complexity_%03d", i))
			if !ok || handler == nil {
				t.Fatalf("missing registered complexity handler for key %d", i)
			}
		}
	})

	t.Run("input unmarshaler", func(t *testing.T) {
		resetInputUnmarshalRegistryForTest()

		type marker struct{ id string }

		const total = 128
		const readers = 8

		errCh := make(chan error, total+readers)
		reportErr := func(err error) {
			select {
			case errCh <- err:
			default:
			}
		}

		start := make(chan struct{})
		var writersWG sync.WaitGroup
		var readersWG sync.WaitGroup
		var writesDone atomic.Bool

		for i := 0; i < total; i++ {
			writersWG.Add(1)
			go func(i int) {
				defer writersWG.Done()
				<-start
				RegisterInputUnmarshaler("scope", fmt.Sprintf("Input_%03d", i), &marker{id: fmt.Sprintf("%03d", i)})
			}(i)
		}

		for i := 0; i < readers; i++ {
			readersWG.Add(1)
			go func() {
				defer readersWG.Done()
				<-start
				for !writesDone.Load() {
					unmarshalers := ListInputUnmarshalers("scope", nil)
					if len(unmarshalers) > total {
						reportErr(fmt.Errorf("unexpected input unmarshaler count: got %d want <= %d", len(unmarshalers), total))
					}
					for i, unmarshaler := range unmarshalers {
						if unmarshaler == nil {
							reportErr(fmt.Errorf("nil input unmarshaler at index %d", i))
						}
					}
				}
			}()
		}

		close(start)
		writersWG.Wait()
		writesDone.Store(true)
		readersWG.Wait()

		close(errCh)
		for err := range errCh {
			t.Fatal(err)
		}

		unmarshalers := ListInputUnmarshalers("scope", nil)
		if len(unmarshalers) != total {
			t.Fatalf("unexpected number of registered input unmarshalers: got %d want %d", len(unmarshalers), total)
		}
		for i, unmarshaler := range unmarshalers {
			if unmarshaler == nil {
				t.Fatalf("nil input unmarshaler at index %d after registration", i)
			}
		}
	})

	t.Run("codec marshal", func(t *testing.T) {
		resetCodecMarshalRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, ast.SelectionSet, any) graphql.Marshaler {
			return graphql.Null
		}

		const total = 128
		const readers = 8

		errCh := make(chan error, total+readers)
		reportErr := func(err error) {
			select {
			case errCh <- err:
			default:
			}
		}

		start := make(chan struct{})
		var writersWG sync.WaitGroup
		var readersWG sync.WaitGroup
		var writesDone atomic.Bool

		for i := 0; i < total; i++ {
			writersWG.Add(1)
			go func(i int) {
				defer writersWG.Done()
				<-start
				RegisterCodecMarshal("scope", fmt.Sprintf("marshal_%03d", i), h)
			}(i)
		}

		for i := 0; i < readers; i++ {
			readersWG.Add(1)
			go func() {
				defer readersWG.Done()
				<-start
				for !writesDone.Load() {
					for i := 0; i < total; i++ {
						handler, ok := LookupCodecMarshal("scope", fmt.Sprintf("marshal_%03d", i))
						if ok && handler == nil {
							reportErr(fmt.Errorf("nil codec marshal handler for registered key %d", i))
						}
					}
				}
			}()
		}

		close(start)
		writersWG.Wait()
		writesDone.Store(true)
		readersWG.Wait()

		close(errCh)
		for err := range errCh {
			t.Fatal(err)
		}

		for i := 0; i < total; i++ {
			handler, ok := LookupCodecMarshal("scope", fmt.Sprintf("marshal_%03d", i))
			if !ok || handler == nil {
				t.Fatalf("missing registered codec marshal handler for key %d", i)
			}
		}
	})

	t.Run("codec unmarshal", func(t *testing.T) {
		resetCodecUnmarshalRegistryForTest()

		h := func(context.Context, ObjectExecutionContext, any) (any, error) {
			return nil, nil
		}

		const total = 128
		const readers = 8

		errCh := make(chan error, total+readers)
		reportErr := func(err error) {
			select {
			case errCh <- err:
			default:
			}
		}

		start := make(chan struct{})
		var writersWG sync.WaitGroup
		var readersWG sync.WaitGroup
		var writesDone atomic.Bool

		for i := 0; i < total; i++ {
			writersWG.Add(1)
			go func(i int) {
				defer writersWG.Done()
				<-start
				RegisterCodecUnmarshal("scope", fmt.Sprintf("unmarshal_%03d", i), h)
			}(i)
		}

		for i := 0; i < readers; i++ {
			readersWG.Add(1)
			go func() {
				defer readersWG.Done()
				<-start
				for !writesDone.Load() {
					for i := 0; i < total; i++ {
						handler, ok := LookupCodecUnmarshal("scope", fmt.Sprintf("unmarshal_%03d", i))
						if ok && handler == nil {
							reportErr(fmt.Errorf("nil codec unmarshal handler for registered key %d", i))
						}
					}
				}
			}()
		}

		close(start)
		writersWG.Wait()
		writesDone.Store(true)
		readersWG.Wait()

		close(errCh)
		for err := range errCh {
			t.Fatal(err)
		}

		for i := 0; i < total; i++ {
			handler, ok := LookupCodecUnmarshal("scope", fmt.Sprintf("unmarshal_%03d", i))
			if !ok || handler == nil {
				t.Fatalf("missing registered codec unmarshal handler for key %d", i)
			}
		}
	})
}

func assertDuplicateRegistrationPanic(t *testing.T, expected string, register func()) {
	t.Helper()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected duplicate registration panic")
		}
		msg, ok := recovered.(string)
		if !ok {
			t.Fatalf("expected panic string, got %T", recovered)
		}
		if msg != expected {
			t.Fatalf("unexpected panic message: got %q want %q", msg, expected)
		}
	}()

	register()
}

func resetObjectRegistryForTest() {
	mu.Lock()
	defer mu.Unlock()

	objectByScope = map[string]map[string]ObjectHandler{}
	resetObjectLookupSnapshotForTest()
}

func resetStreamObjectRegistryForTest() {
	mu.Lock()
	defer mu.Unlock()

	streamByScope = map[string]map[string]StreamObjectHandler{}
	resetStreamObjectLookupSnapshotForTest()
}

func resetFieldRegistryForTest() {
	mu.Lock()
	defer mu.Unlock()

	fieldByScope = map[string]map[string]map[string]FieldHandler{}
	resetFieldLookupSnapshotForTest()
}

func resetStreamFieldRegistryForTest() {
	mu.Lock()
	defer mu.Unlock()

	streamFieldByScope = map[string]map[string]map[string]StreamFieldHandler{}
	resetStreamFieldLookupSnapshotForTest()
}

func resetComplexityRegistryForTest() {
	mu.Lock()
	defer mu.Unlock()

	complexityByScope = map[string]map[string]map[string]ComplexityHandler{}
	resetComplexityLookupSnapshotForTest()
}

func resetInputUnmarshalRegistryForTest() {
	mu.Lock()
	defer mu.Unlock()

	inputUnmarshalByScope = map[string]map[string]any{}
	resetInputUnmarshalLookupSnapshotForTest()
}

func resetCodecMarshalRegistryForTest() {
	mu.Lock()
	defer mu.Unlock()

	codecMarshalByScope = map[string]map[string]CodecMarshalHandler{}
	resetCodecMarshalLookupSnapshotForTest()
}

func resetCodecUnmarshalRegistryForTest() {
	mu.Lock()
	defer mu.Unlock()

	codecUnmarshalByScope = map[string]map[string]CodecUnmarshalHandler{}
	resetCodecUnmarshalLookupSnapshotForTest()
}

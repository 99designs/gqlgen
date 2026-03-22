package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func TestAddFieldLocationToError_AddsLocations(t *testing.T) {
	ctx := context.Background()
	ctx = WithFieldContext(ctx, &FieldContext{
		Field: CollectedField{
			Field: &ast.Field{
				Name: "customer",
				Position: &ast.Position{
					Line:   1,
					Column: 3,
				},
			},
		},
	})

	err := AddFieldLocationToError(ctx, errors.New("not authorized"))

	var gqlErr *gqlerror.Error
	if !errors.As(err, &gqlErr) {
		t.Fatal("expected gqlerror.Error")
	}

	if len(gqlErr.Locations) == 0 {
		t.Fatal("expected locations to be set on resolver error")
	}
	if gqlErr.Locations[0].Line != 1 {
		t.Errorf("expected line 1, got %d", gqlErr.Locations[0].Line)
	}
	if gqlErr.Locations[0].Column != 3 {
		t.Errorf("expected column 3, got %d", gqlErr.Locations[0].Column)
	}
}

func TestAddFieldLocationToError_PreservesExistingLocations(t *testing.T) {
	ctx := context.Background()
	ctx = WithFieldContext(ctx, &FieldContext{
		Field: CollectedField{
			Field: &ast.Field{
				Name:     "customer",
				Position: &ast.Position{Line: 1, Column: 3},
			},
		},
	})

	existing := &gqlerror.Error{
		Message:   "existing error",
		Locations: []gqlerror.Location{{Line: 5, Column: 10}},
	}

	err := AddFieldLocationToError(ctx, existing)

	var gqlErr *gqlerror.Error
	if !errors.As(err, &gqlErr) {
		t.Fatal("expected gqlerror.Error")
	}
	if len(gqlErr.Locations) != 1 || gqlErr.Locations[0].Line != 5 {
		t.Errorf("expected existing location (line 5) to be preserved, got %v", gqlErr.Locations)
	}
}

func TestAddFieldLocationToError_NoFieldContext(t *testing.T) {
	ctx := context.Background()

	origErr := errors.New("some error")
	err := AddFieldLocationToError(ctx, origErr)

	// Should return original error unchanged
	if err != origErr {
		t.Error("expected original error when FieldContext is missing")
	}
}

func TestAddFieldLocationToError_NilPosition(t *testing.T) {
	ctx := context.Background()
	ctx = WithFieldContext(ctx, &FieldContext{
		Field: CollectedField{
			Field: &ast.Field{
				Name:     "customer",
				Position: nil,
			},
		},
	})

	origErr := errors.New("some error")
	err := AddFieldLocationToError(ctx, origErr)

	if err != origErr {
		t.Error("expected original error when Position is nil")
	}
}

func TestAddFieldLocationToError_NilError(t *testing.T) {
	result := AddFieldLocationToError(context.Background(), nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}
}

func TestErrorOnPath_Unchanged(t *testing.T) {
	// Verify ErrorOnPath still works as before (no locations added)
	ctx := context.Background()
	ctx = WithFieldContext(ctx, &FieldContext{
		Field: CollectedField{
			Field: &ast.Field{
				Name:     "customer",
				Position: &ast.Position{Line: 1, Column: 3},
			},
		},
	})

	err := ErrorOnPath(ctx, errors.New("some error"))

	var gqlErr *gqlerror.Error
	if !errors.As(err, &gqlErr) {
		t.Fatal("expected gqlerror.Error")
	}
	if gqlErr.Locations != nil {
		t.Error("ErrorOnPath should NOT add locations (only AddFieldLocationToError does)")
	}
}

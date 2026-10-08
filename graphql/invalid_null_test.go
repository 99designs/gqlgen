package graphql

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func TestInvalidNullErrorMessage(t *testing.T) {
	specs := []struct {
		Name     string
		Err      *InvalidNullError
		Expected string
	}{
		{
			Name: "field",
			Err: &InvalidNullError{
				Object:       "OuterObject",
				Field:        "inner",
				DeclaredType: "InnerObject!",
				Source:       NullSourceModelField,
			},
			Expected: "cannot return null for non-null field OuterObject.inner (InnerObject!): " +
				"the model struct field was nil and the field has no resolver",
		},
		{
			Name: "list element",
			Err: &InvalidNullError{
				Object:       "Query",
				Field:        "errorBubbleList",
				DeclaredType: "Error!",
				Source:       NullSourceResolver,
				ListElement:  true,
			},
			Expected: "cannot return null for non-null element of field Query.errorBubbleList " +
				"(Error!): the list element was nil",
		},
		{
			Name: "resolver",
			Err: &InvalidNullError{
				Object:       "Errors",
				Field:        "b",
				DeclaredType: "Error!",
				Source:       NullSourceResolver,
			},
			Expected: "cannot return null for non-null field Errors.b (Error!): " +
				"the resolver returned nil",
		},
		{
			Name: "model method",
			Err: &InvalidNullError{
				Object:       "ConcreteNodeA",
				Field:        "child",
				DeclaredType: "Node!",
				Source:       NullSourceModelMethod,
			},
			Expected: "cannot return null for non-null field ConcreteNodeA.child (Node!): " +
				"the model method returned nil",
		},
		{
			Name: "marked non-null at runtime",
			Err: &InvalidNullError{
				Object:       "Query",
				Field:        "maybe",
				DeclaredType: "String",
				Source:       NullSourceRuntimeNonNull,
			},
			Expected: "cannot return null for non-null field Query.maybe (String): " +
				"the field was marked non-null at runtime and resolved to nil",
		},
		{
			Name: "marshaler outranks the position of the null",
			Err: &InvalidNullError{
				Object:       "User",
				Field:        "created",
				DeclaredType: "Time!",
				Source:       NullSourceMarshaler,
			},
			Expected: "cannot return null for non-null field User.created (Time!): " +
				"the marshaler returned null for a non-nil value",
		},
		{
			Name: "field with no owning type still names the field",
			Err: &InvalidNullError{
				Field:        "inner",
				DeclaredType: "InnerObject!",
				Source:       NullSourceResolver,
			},
			Expected: "cannot return null for non-null field inner (InnerObject!): " +
				"the resolver returned nil",
		},
		{
			Name: "no field name falls back to the response path",
			Err: &InvalidNullError{
				DeclaredType: "InnerObject!",
				Source:       NullSourceUnknown,
				Path: ast.Path{
					ast.PathName("nestedOutputs"), ast.PathIndex(0), ast.PathIndex(0),
					ast.PathName("inner"),
				},
			},
			Expected: "cannot return null for non-null response position " +
				"nestedOutputs[0][0].inner (InnerObject!): the value was nil",
		},
		{
			Name:     "no field context, type known",
			Err:      &InvalidNullError{DeclaredType: "String!", Source: NullSourceUnknown},
			Expected: "cannot return null for non-null position (String!): the value was nil",
		},
		{
			Name:     "nothing known",
			Err:      &InvalidNullError{Source: NullSourceUnknown},
			Expected: "cannot return null for non-null position: the value was nil",
		},
	}

	for _, spec := range specs {
		t.Run(spec.Name, func(t *testing.T) {
			assert.Equal(t, spec.Expected, spec.Err.Error())
		})
	}
}

func TestInvalidNullErrorSchemaCoordinate(t *testing.T) {
	specs := []struct {
		Name     string
		Err      InvalidNullError
		Expected string
	}{
		{
			Name:     "object and field",
			Err:      InvalidNullError{Object: "User", Field: "name"},
			Expected: "User.name",
		},
		{Name: "field with no object", Err: InvalidNullError{Field: "name"}, Expected: ""},
		{Name: "object with no field", Err: InvalidNullError{Object: "User"}, Expected: ""},
	}

	for _, spec := range specs {
		t.Run(spec.Name, func(t *testing.T) {
			assert.Equal(t, spec.Expected, spec.Err.SchemaCoordinate())
		})
	}
}

// TestInvalidNullErrorMatching covers identifying the error without its message.
func TestInvalidNullErrorMatching(t *testing.T) {
	err := &InvalidNullError{Object: "Errors", Field: "b", DeclaredType: "Error!"}

	t.Run("errors.Is finds the sentinel", func(t *testing.T) {
		require.ErrorIs(t, err, ErrInvalidNull)
	})

	t.Run("the sentinel stays out of the unwrap chain", func(t *testing.T) {
		// The sentinel caused nothing, so no chain walker should see it.
		assert.NoError(t, errors.Unwrap(error(err)))
	})

	t.Run("errors.AsType recovers it through a gqlerror wrap", func(t *testing.T) {
		// The shape a custom ErrorPresenter receives, via ErrorOnPath.
		wrapped := gqlerror.WrapPath(ast.Path{ast.PathName("errors")}, error(err))

		require.ErrorIs(t, wrapped, ErrInvalidNull)
		got, ok := errors.AsType[*InvalidNullError](error(wrapped))
		require.True(t, ok)
		assert.Equal(t, "Errors.b", got.SchemaCoordinate())
		assert.Equal(t, "Error!", got.DeclaredType)
	})
}

// TestAddInvalidNullErrorClassification drives the exported entry point with the
// field-context chains the executor builds. Chains are declared root-first;
// WithFieldContext links each to the one before, as generated code does.
func TestAddInvalidNullErrorClassification(t *testing.T) {
	// Query.errorBubbleList: [Error!], resolved by a user resolver.
	listField := func() *FieldContext {
		return &FieldContext{
			Object:     "Query",
			IsMethod:   true,
			IsResolver: true,
			Field: CollectedField{Field: &ast.Field{
				Name:  "errorBubbleList",
				Alias: "errorBubbleList",
				Definition: &ast.FieldDefinition{
					Type: ast.ListType(ast.NonNullNamedType("Error", nil), nil),
				},
			}},
		}
	}
	index := 1

	specs := []struct {
		Name         string
		Chain        []*FieldContext
		DeclaredType string
		Marshaler    bool // call AddInvalidNullFromMarshaler instead
		Expected     InvalidNullError
	}{
		{
			Name:  "resolver",
			Chain: []*FieldContext{listField()},
			Expected: InvalidNullError{
				Object:       "Query",
				Field:        "errorBubbleList",
				DeclaredType: "[Error!]",
				Source:       NullSourceResolver,
				Path:         ast.Path{ast.PathName("errorBubbleList")},
			},
		},
		{
			Name:         "list element walks up to the naming field",
			Chain:        []*FieldContext{listField(), {Index: &index}},
			DeclaredType: "Error!",
			Expected: InvalidNullError{
				Object:       "Query",
				Field:        "errorBubbleList",
				DeclaredType: "Error!",
				Source:       NullSourceResolver,
				ListElement:  true,
				Path:         ast.Path{ast.PathName("errorBubbleList"), ast.PathIndex(1)},
			},
		},
		{
			Name: "model method",
			Chain: []*FieldContext{{
				Object:   "ConcreteNodeA",
				IsMethod: true,
				Field: CollectedField{Field: &ast.Field{
					Name:       "child",
					Alias:      "child",
					Definition: &ast.FieldDefinition{Type: ast.NonNullNamedType("Node", nil)},
				}},
			}},
			Expected: InvalidNullError{
				Object:       "ConcreteNodeA",
				Field:        "child",
				DeclaredType: "Node!",
				Source:       NullSourceModelMethod,
				Path:         ast.Path{ast.PathName("child")},
			},
		},
		{
			Name: "model struct field",
			Chain: []*FieldContext{{
				Object: "OuterObject",
				Field: CollectedField{Field: &ast.Field{
					Name:  "inner",
					Alias: "inner",
					Definition: &ast.FieldDefinition{
						Type: ast.NonNullNamedType("InnerObject", nil),
					},
				}},
			}},
			Expected: InvalidNullError{
				Object:       "OuterObject",
				Field:        "inner",
				DeclaredType: "InnerObject!",
				Source:       NullSourceModelField,
				Path:         ast.Path{ast.PathName("inner")},
			},
		},
		{
			Name: "nullable field marked non-null at runtime",
			Chain: []*FieldContext{{
				Object:   "Query",
				NonNull:  true,
				IsMethod: true,
				Field: CollectedField{Field: &ast.Field{
					Name:       "maybe",
					Alias:      "maybe",
					Definition: &ast.FieldDefinition{Type: ast.NamedType("String", nil)},
				}},
			}},
			Expected: InvalidNullError{
				Object:       "Query",
				Field:        "maybe",
				DeclaredType: "String",
				Source:       NullSourceRuntimeNonNull,
				Path:         ast.Path{ast.PathName("maybe")},
			},
		},
		{
			Name: "marshaler nulls a present value",
			Chain: []*FieldContext{{
				Object: "User",
				Field: CollectedField{Field: &ast.Field{
					Name:       "created",
					Alias:      "created",
					Definition: &ast.FieldDefinition{Type: ast.NonNullNamedType("Time", nil)},
				}},
			}},
			DeclaredType: "Time!",
			Marshaler:    true,
			Expected: InvalidNullError{
				Object:       "User",
				Field:        "created",
				DeclaredType: "Time!",
				Source:       NullSourceMarshaler,
				Path:         ast.Path{ast.PathName("created")},
			},
		},
		{
			// A coordinate names the schema, so an aliased field still reports
			// its schema name. The response path uses the alias.
			Name: "alias does not replace the schema field name",
			Chain: []*FieldContext{{
				Object:   "OuterObject",
				IsMethod: true,
				Field: CollectedField{Field: &ast.Field{
					Name:  "inner",
					Alias: "renamed",
					Definition: &ast.FieldDefinition{
						Type: ast.NonNullNamedType("InnerObject", nil),
					},
				}},
			}},
			Expected: InvalidNullError{
				Object:       "OuterObject",
				Field:        "inner",
				DeclaredType: "InnerObject!",
				Source:       NullSourceModelMethod,
				Path:         ast.Path{ast.PathName("renamed")},
			},
		},
		{
			Name:         "no field context",
			DeclaredType: "String!",
			Expected: InvalidNullError{
				DeclaredType: "String!",
				Source:       NullSourceUnknown,
			},
		},
		{
			// An index with nothing above it names no field, but it is still a
			// list element, which is what the message leans on.
			Name:         "index with no naming ancestor",
			Chain:        []*FieldContext{{Index: &index}},
			DeclaredType: "Error!",
			Expected: InvalidNullError{
				DeclaredType: "Error!",
				Source:       NullSourceUnknown,
				ListElement:  true,
				Path:         ast.Path{ast.PathIndex(1)},
			},
		},
		{
			// object.gotpl pushes this for root types. Walking past a context
			// that merely lacks a name must not imply a list.
			Name:         "root-object context is not a list element",
			Chain:        []*FieldContext{{Object: "Query"}},
			DeclaredType: "Error!",
			Expected: InvalidNullError{
				DeclaredType: "Error!",
				Source:       NullSourceUnknown,
			},
		},
	}

	for _, spec := range specs {
		t.Run(spec.Name, func(t *testing.T) {
			ctx := WithResponseContext(context.Background(), DefaultErrorPresenter, nil)
			for _, fc := range spec.Chain {
				ctx = WithFieldContext(ctx, fc)
			}

			if spec.Marshaler {
				AddInvalidNullFromMarshaler(ctx, spec.DeclaredType)
			} else {
				AddInvalidNullError(ctx, spec.DeclaredType)
			}

			errs := GetErrors(ctx)
			require.Len(t, errs, 1)
			got, ok := errors.AsType[*InvalidNullError](error(errs[0]))
			require.True(t, ok)
			assert.Equal(t, &spec.Expected, got)
		})
	}
}

// TestAddInvalidNullErrorSkipsAlreadyErroredField covers the deduplication the spec
// asks for: the earlier error at this path is the cause of the null.
func TestAddInvalidNullErrorSkipsAlreadyErroredField(t *testing.T) {
	fc := &FieldContext{
		Object:     "Error",
		IsMethod:   true,
		IsResolver: true,
		Field: CollectedField{Field: &ast.Field{
			Name:       "errorOnRequiredField",
			Alias:      "errorOnRequiredField",
			Definition: &ast.FieldDefinition{Type: ast.NonNullNamedType("String", nil)},
		}},
	}

	t.Run("adds when the field is clean", func(t *testing.T) {
		ctx := WithFieldContext(
			WithResponseContext(context.Background(), DefaultErrorPresenter, nil),
			fc,
		)
		AddInvalidNullError(ctx, "String!")

		errs := GetErrors(ctx)
		require.Len(t, errs, 1)
		got, ok := errors.AsType[*InvalidNullError](error(errs[0]))
		require.True(t, ok)
		assert.Equal(t, "Error.errorOnRequiredField", got.SchemaCoordinate())
	})

	t.Run("skips when the field already errored", func(t *testing.T) {
		ctx := WithFieldContext(
			WithResponseContext(context.Background(), DefaultErrorPresenter, nil),
			fc,
		)
		AddError(ctx, errors.New("boom"))
		AddInvalidNullError(ctx, "String!")

		errs := GetErrors(ctx)
		require.Len(t, errs, 1)
		assert.Equal(t, "boom", errs[0].Message)
	})
}

func TestInvalidNullErrorAttrs(t *testing.T) {
	specs := []struct {
		Name     string
		Err      InvalidNullError
		Expected []slog.Attr
	}{
		{
			Name: "field",
			Err: InvalidNullError{
				Object:       "Errors",
				Field:        "b",
				DeclaredType: "Error!",
				Source:       NullSourceResolver,
				Path:         ast.Path{ast.PathName("errors"), ast.PathName("b")},
			},
			Expected: []slog.Attr{
				slog.Group("gql",
					slog.String("field", "Errors.b"),
					slog.String("type", "Error!"),
					slog.String("null_source", "resolver"),
					slog.String("path", "errors.b"),
				),
			},
		},
		{
			Name: "list element carries the flag and an indexed path",
			Err: InvalidNullError{
				Object:       "Query",
				Field:        "errorBubbleList",
				DeclaredType: "Error!",
				Source:       NullSourceResolver,
				ListElement:  true,
				Path:         ast.Path{ast.PathName("errorBubbleList"), ast.PathIndex(1)},
			},
			Expected: []slog.Attr{
				slog.Group("gql",
					slog.String("field", "Query.errorBubbleList"),
					slog.String("type", "Error!"),
					slog.String("null_source", "resolver"),
					slog.Bool("list_element", true),
					slog.String("path", "errorBubbleList[1]"),
				),
			},
		},
		{
			Name:     "nothing known yields no blanks",
			Err:      InvalidNullError{},
			Expected: nil,
		},
	}

	for _, spec := range specs {
		t.Run(spec.Name, func(t *testing.T) {
			assert.Equal(t, spec.Expected, spec.Err.Attrs())
		})
	}
}

// TestInvalidNullErrorAttrsThroughPresenter checks the attributes survive the error
// presenter and the GetErrors copy, reached by asking the chain for Attrs()
// rather than for a gqlgen type.
func TestInvalidNullErrorAttrsThroughPresenter(t *testing.T) {
	ctx := WithFieldContext(
		WithResponseContext(context.Background(), DefaultErrorPresenter, nil),
		&FieldContext{
			Object:     "Errors",
			IsMethod:   true,
			IsResolver: true,
			Field: CollectedField{Field: &ast.Field{
				Name:       "b",
				Alias:      "b",
				Definition: &ast.FieldDefinition{Type: ast.NonNullNamedType("Error", nil)},
			}},
		},
	)
	AddInvalidNullError(ctx, "Error!")

	errs := GetErrors(ctx)
	require.Len(t, errs, 1)

	// Both errors report into "gql"; CollectAttrs merges them, keeping the outer path.
	assert.Equal(t, []slog.Attr{
		slog.Group("gql",
			slog.String("message", errs[0].Message),
			slog.String("path", "b"),
			slog.String("field", "Errors.b"),
			slog.String("type", "Error!"),
			slog.String("null_source", "resolver"),
		),
	}, gqlerror.CollectAttrs(errs[0]))
}

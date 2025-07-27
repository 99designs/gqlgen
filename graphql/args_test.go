package graphql

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

func TestProcessArgField(t *testing.T) {
	tests := []struct {
		name          string
		rawArgs       map[string]any
		fieldName     string
		valueMapperFn func(ctx context.Context, value any) (any, error)
		expected      any
		expectedErr   string
	}{
		{
			name:      "field does not exist",
			rawArgs:   map[string]any{},
			fieldName: "name",
			valueMapperFn: func(ctx context.Context, value any) (any, error) {
				return "", errors.New("should not be called")
			},
		},
		{
			name:      "field exists",
			rawArgs:   map[string]any{"name": "test"},
			fieldName: "name",
			valueMapperFn: func(ctx context.Context, value any) (any, error) {
				path := GetPath(ctx)
				assert.Equal(t, ast.Path{ast.PathName("name")}, path)
				return value.(string), nil
			},
			expected: "test",
		},
		{
			name:      "valueMapperFn returns an error",
			rawArgs:   map[string]any{"name": "test"},
			fieldName: "name",
			valueMapperFn: func(ctx context.Context, value any) (any, error) {
				return nil, errors.New("mapper error")
			},
			expectedErr: "mapper error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ProcessArgField(context.Background(), test.rawArgs, test.fieldName, test.valueMapperFn)
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			}
		})
	}
}

func TestProcessArgFieldWithEC(t *testing.T) {
	type executionContext struct {
		someValue string
	}

	tests := []struct {
		name          string
		ec            *executionContext
		rawArgs       map[string]any
		fieldName     string
		valueMapperFn func(ctx context.Context, ec *executionContext, value any) (any, error)
		expected      any
		expectedErr   string
	}{
		{
			name:      "field does not exist",
			ec:        &executionContext{someValue: "test1"},
			rawArgs:   map[string]any{},
			fieldName: "name",
			valueMapperFn: func(ctx context.Context, ec *executionContext, value any) (any, error) {
				return "", errors.New("should not be called")
			},
		},
		{
			name:      "field exists",
			ec:        &executionContext{someValue: "test2"},
			rawArgs:   map[string]any{"name": "test"},
			fieldName: "name",
			valueMapperFn: func(ctx context.Context, ec *executionContext, value any) (any, error) {
				path := GetPath(ctx)
				assert.Equal(t, "test2", ec.someValue)
				assert.Equal(t, ast.Path{ast.PathName("name")}, path)
				return value.(string), nil
			},
			expected: "test",
		},
		{
			name:      "valueMapperFn returns an error",
			ec:        &executionContext{someValue: "test3"},
			rawArgs:   map[string]any{"name": "test"},
			fieldName: "name",
			valueMapperFn: func(ctx context.Context, ec *executionContext, value any) (any, error) {
				assert.Equal(t, "test3", ec.someValue)
				return nil, errors.New("mapper error")
			},
			expectedErr: "mapper error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ProcessArgFieldWithEC(context.Background(), test.ec, test.rawArgs, test.fieldName, test.valueMapperFn)
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			}
		})
	}
}

package graphql

import (
	"context"
)

// ProcessArgField Parses argument value without Execution Context
// This function is called from generated code
func ProcessArgField[T any](
	ctx context.Context,
	rawArgs map[string]any,
	fieldName string,
	valueMapperFn func(ctx context.Context, value any) (T, error),
) (T, error) {
	value, exists := rawArgs[fieldName]
	if !exists {
		var zeroVal T
		return zeroVal, nil
	}

	ctx = WithPathContext(ctx, NewPathWithField(fieldName))
	return valueMapperFn(ctx, value)
}

// ProcessArgFieldWithEC Parses argument value with Execution Context
// This function is called from generated code
func ProcessArgFieldWithEC[T, EC any](
	ctx context.Context,
	ec EC,
	rawArgs map[string]any,
	fieldName string,
	valueMapperFn func(ctx context.Context, ec EC, value any) (T, error),
) (T, error) {
	value, exists := rawArgs[fieldName]
	if !exists {
		var zeroVal T
		return zeroVal, nil
	}

	ctx = WithPathContext(ctx, NewPathWithField(fieldName))
	return valueMapperFn(ctx, ec, value)
}

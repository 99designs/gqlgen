package graphql

import (
	"context"
	"errors"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

type ErrorPresenterFunc func(ctx context.Context, err error) *gqlerror.Error

func DefaultErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	if err == nil {
		return nil
	}
	var gqlErr *gqlerror.Error
	if errors.As(err, &gqlErr) {
		return gqlErr
	}
	return gqlerror.WrapPath(GetPath(ctx), err)
}

func ErrorOnPath(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	var gqlErr *gqlerror.Error
	if errors.As(err, &gqlErr) {
		if gqlErr.Path == nil {
			gqlErr.Path = GetPath(ctx)
		}
		// Return the original error to avoid losing any attached annotation
		return err
	}
	return gqlerror.WrapPath(GetPath(ctx), err)
}

// AddFieldLocationToError attaches the source location of the current field
// to a resolver error. Called from resolveField after a resolver returns an error.
// This ensures resolver errors include the locations field as required by the
// GraphQL spec (https://spec.graphql.org/October2021/#sec-Errors).
func AddFieldLocationToError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	fc := GetFieldContext(ctx)
	if fc == nil || fc.Field.Position == nil {
		return err
	}
	pos := fc.Field.Position
	if pos.Line == 0 {
		return err
	}

	loc := gqlerror.Location{Line: pos.Line, Column: pos.Column}

	var gqlErr *gqlerror.Error
	if errors.As(err, &gqlErr) {
		if gqlErr.Locations == nil {
			gqlErr.Locations = []gqlerror.Location{loc}
		}
		return err
	}

	// Wrap non-gqlerror errors
	wrapped := gqlerror.WrapPath(GetPath(ctx), err)
	wrapped.Locations = []gqlerror.Location{loc}
	return wrapped
}

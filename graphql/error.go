package graphql

import (
	"context"
	"errors"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

type ErrorPresenterFunc func(ctx context.Context, err error) *gqlerror.Error

type ExtendedError interface {
	Extensions() map[string]interface{}
}

func DefaultErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	var gqlerr *gqlerror.Error
	if errors.As(err, &gqlerr) {
		if gqlerr.Path == nil {
			gqlerr.Path = GetFieldContext(ctx).Path()
		}
		return gqlerr
	}

	gqlerr = &gqlerror.Error{
		Message: err.Error(),
		Path:    GetFieldContext(ctx).Path(),
	}
	var ee ExtendedError
	if errors.As(err, &ee) {
		gqlerr.Extensions = ee.Extensions()
	}
	return gqlerr
}

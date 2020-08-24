package graphql

import (
	"context"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type ErrorPresenterFunc func(ctx context.Context, err error) *gqlerror.Error

type ExtendedError interface {
	Extensions() map[string]interface{}
}

func DefaultErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	if gqlerr, ok := err.(*gqlerror.Error); ok {
		if gqlerr.Path == nil {
			gqlerr.Path = GetPathFromContext(ctx)
		}
		return gqlerr
	}

	var extensions map[string]interface{}
	if ee, ok := err.(ExtendedError); ok {
		extensions = ee.Extensions()
	}

	return &gqlerror.Error{
		Message:    err.Error(),
		Path:       GetPathFromContext(ctx),
		Extensions: extensions,
	}
}

func GetPathFromContext(ctx context.Context) ast.Path {
	if in := GetFieldInputContext(ctx); in != nil {
		return in.Path()
	}
	return GetFieldContext(ctx).Path()
}

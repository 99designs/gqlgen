package graphql

import (
	"context"

	"github.com/vektah/gqlgen/neelance/errors"
	"github.com/vektah/gqlgen/neelance/query"
)

type RequestContext struct {
	errors.Builder

	Variables map[string]interface{}
	Doc       *query.Document
	Recover   RecoverFunc
}

type key string

const rcKey key = "request_context"

func GetRequestContext(ctx context.Context) *RequestContext {
	val := ctx.Value(rcKey)
	if val == nil {
		return nil
	}

	return val.(*RequestContext)
}

func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, rcKey, rc)
}

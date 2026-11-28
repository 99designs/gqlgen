package execbehavior

import (
	"context"
	"errors"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
)

// MarshalMaybe marshals v, and the empty string as null.
func MarshalMaybe(v string) graphql.Marshaler {
	if v == "" {
		return graphql.Null
	}
	return graphql.MarshalString(v)
}

// UnmarshalMaybe unmarshals a string.
func UnmarshalMaybe(v any) (string, error) {
	return graphql.UnmarshalString(v)
}

// MarshalCtxMaybe marshals v with the name of the field, and the empty string as null.
func MarshalCtxMaybe(v string) graphql.ContextMarshaler {
	if v == "" {
		return graphql.Null
	}
	return graphql.ContextWriterFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, strconv.Quote(graphql.GetFieldContext(ctx).Field.Name+":"+v))
		return err
	})
}

// UnmarshalCtxMaybe unmarshals a string, and fails for "bad".
func UnmarshalCtxMaybe(ctx context.Context, v any) (string, error) {
	s, err := graphql.UnmarshalString(v)
	if err != nil {
		return "", err
	}
	if s == "bad" {
		return "", errors.New("bad CtxMaybe")
	}
	return s, nil
}

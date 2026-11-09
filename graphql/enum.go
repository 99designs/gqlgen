package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"slices"
)

// MarshalEnum eagerly serializes an enum and checks its result against the schema's
// allowed values. On failure it records a field error and returns Null so generated
// callers can propagate non-null failures before writing the response. Context
// marshalers are invoked directly to preserve their returned errors. Successful
// results are buffered, so the original marshaler is invoked only once.
func MarshalEnum(ctx context.Context, name string, value Marshaler, allowed ...string) Marshaler {
	var buf bytes.Buffer
	if contextual, ok := value.(ContextMarshaler); ok {
		if err := contextual.MarshalGQLContext(ctx, &buf); err != nil {
			AddError(ctx, err)
			return Null
		}
	} else {
		value.MarshalGQL(&buf)
	}

	var result *string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil || result == nil ||
		!slices.Contains(allowed, *result) {
		AddErrorf(ctx, "invalid value for enum %s", name)
		return Null
	}
	return WriterFunc(func(w io.Writer) {
		w.Write(buf.Bytes())
	})
}

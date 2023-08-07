package graphql

import (
	"fmt"
	"io"

	"github.com/google/uuid"
)

func MarshalUUID(id uuid.UUID) Marshaler {
	return WriterFunc(func(w io.Writer) {
		writeQuotedString(w, id.String())
	})
}

func UnmarshalUUID(v any) (uuid.UUID, error) {
	switch v := v.(type) {
	case string:
		return uuid.Parse(v)
	case []byte:
		return uuid.ParseBytes(v)
	default:
		return uuid.Nil, fmt.Errorf("%T is not a uuid", v)
	}
}

package graphql

import (
	"errors"
	"io"

	"github.com/gofrs/uuid"
)

func MarshalUUID(t uuid.UUID) Marshaler {
	if t.IsNil() {
		return Null
	}
	return WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, t.String())
	})
}

func UnmarshalUUID(v interface{}) (uuid.UUID, error) {
	if str, ok := v.(string); ok {
		return uuid.FromString(str)
	}
	return uuid.Nil, errors.New("input must be an RFC-4122 formatted string")
}

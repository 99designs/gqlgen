package graphql

import (
	"fmt"
	"io"
	"strconv"

	"github.com/google/uuid"
)

// MarshalUUID returns the string form of uuid, xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func MarshalUUID(u uuid.UUID) Marshaler {

	return WriterFunc(
		func(w io.Writer) {
			io.WriteString(w, strconv.Quote(u.String()))
		},
	)
}

func UnmarshalUUID(v interface{}) (uuid.UUID, error) {
	switch v := v.(type) {
	case string:
		uid, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, fmt.Errorf("%T is not an UUID: %w", v, err)
		}
		return uid, nil
	default:
		return uuid.Nil, fmt.Errorf("%T is not an UUID", v)
	}
}

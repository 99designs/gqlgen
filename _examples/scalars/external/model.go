package external

import (
	"fmt"
	"io"

	"github.com/99designs/gqlgen/graphql"
)

type (
	ObjectID      int
	Manufacturer  string // remote named string
	Count         uint8  // remote named uint8
	ExternalBytes []byte
	ExternalRunes []rune
)

const (
	ManufacturerTesla  Manufacturer = "TESLA"
	ManufacturerHonda  Manufacturer = "HONDA"
	ManufacturerToyota Manufacturer = "TOYOTA"
)

func MarshalBytes(b ExternalBytes) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "%q", string(b))
	})
}

func UnmarshalBytes(v interface{}) (ExternalBytes, error) {
	switch v := v.(type) {
	case string:
		return ExternalBytes(v), nil
	case *string:
		return ExternalBytes(*v), nil
	case ExternalBytes:
		return v, nil
	default:
		return nil, fmt.Errorf("%T is not ExternalBytes", v)
	}
}

func MarshalRunes(r ExternalRunes) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = fmt.Fprintf(w, "%q", string(r))
	})
}

func UnmarshalRunes(v interface{}) (ExternalRunes, error) {
	switch v := v.(type) {
	case string:
		return ExternalRunes(v), nil
	case *string:
		return ExternalRunes(*v), nil
	case ExternalRunes:
		return v, nil
	default:
		return nil, fmt.Errorf("%T is not ExternalRunes", v)
	}
}

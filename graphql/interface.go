package graphql

import (
	"encoding/json"
	"io"

	"github.com/qhenkart/gqlgen/graphql"
)

func MarshalInterface(v interface{}) Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		err := json.NewEncoder(w).Encode(val)
		if err != nil {
			panic(err)
		}
	})
}

func UnmarshalInterface(v interface{}) (interface{}, error) {
	return v, nil
}

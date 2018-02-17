package graphql

import (
	"io"

	"github.com/vektah/gqlgen/neelance/errors"
)

type Response struct {
	Data   Marshaler
	Errors []*errors.QueryError
}

func (r *Response) MarshalGQL(w io.Writer) {
	result := &OrderedMap{}
	if r.Data == nil {
		result.Add("data", Null)
	} else {
		result.Add("data", r.Data)
	}

	if len(r.Errors) > 0 {
		result.Add("errors", MarshalErrors(r.Errors))
	}

	result.MarshalGQL(w)
}

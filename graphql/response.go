package graphql

import (
	"encoding/json"

	"github.com/vektah/gqlgen/neelance/errors"
)

type Response struct {
	Data   json.RawMessage      `json:"data"`
	Errors []*errors.QueryError `json:"errors,omitempty"`
}

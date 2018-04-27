package graphql

import (
	"context"
	"encoding/json"
	"fmt"
)

type Response struct {
	Data   json.RawMessage `json:"data"`
	Errors []error         `json:"errors,omitempty"`
}

func ErrorResponse(ctx context.Context, messagef string, args ...interface{}) *Response {
	return &Response{
		Errors: []error{&ResolverError{Message: fmt.Sprintf(messagef, args...)}},
	}
}

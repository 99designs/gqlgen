package graphql

import (
	"context"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/exec"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

type Schema struct {
	exec *exec.Exec
}

func ParseSchema(schemaString string, resolver interface{}) (*Schema, error) {
	s, err := schema.Parse(schemaString)
	if err != nil {
		return nil, err
	}

	e, err2 := exec.Make(s, resolver)
	if err2 != nil {
		return nil, err2
	}
	return &Schema{
		exec: e,
	}, nil
}

type Response struct {
	Data       interface{}            `json:"data,omitempty"`
	Errors     []*errors.GraphQLError `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func (s *Schema) Exec(ctx context.Context, queryString string, operationName string, variables map[string]interface{}) *Response {
	d, err := query.Parse(queryString)
	if err != nil {
		return &Response{
			Errors: []*errors.GraphQLError{err},
		}
	}

	if operationName == "" && len(d.Operations) == 1 {
		for name := range d.Operations {
			operationName = name
		}
	}

	op, ok := d.Operations[operationName]
	if !ok {
		return &Response{
			Errors: []*errors.GraphQLError{errors.Errorf("no operation with name %q", operationName)},
		}
	}

	data, errs := s.exec.Exec(ctx, d, variables, op.SelSet)
	return &Response{
		Data:   data,
		Errors: errs,
	}
}

package graphql

import (
	"context"
	"encoding/json"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

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

	if len(d.Operations) == 0 {
		return &Response{
			Errors: []*errors.GraphQLError{errors.Errorf("no operations in query document")},
		}
	}

	var op *query.Operation
	if operationName == "" {
		if len(d.Operations) > 1 {
			return &Response{
				Errors: []*errors.GraphQLError{errors.Errorf("more than one operation in query document and no operation name given")},
			}
		}
		for _, op2 := range d.Operations {
			op = op2
		}
	} else {
		var ok bool
		op, ok = d.Operations[operationName]
		if !ok {
			return &Response{
				Errors: []*errors.GraphQLError{errors.Errorf("no operation with name %q", operationName)},
			}
		}
	}

	span, subCtx := opentracing.StartSpanFromContext(ctx, "GraphQL request")
	span.SetTag("query", queryString)
	if operationName != "" {
		span.SetTag("operationName", operationName)
	}
	if len(variables) != 0 {
		span.SetTag("variables", variables)
	}
	defer span.Finish()

	data, errs := s.exec.Exec(subCtx, d, variables, op)
	if len(errs) != 0 {
		ext.Error.Set(span, true)
		span.SetTag("errorMsg", errs)
	}
	return &Response{
		Data:   data,
		Errors: errs,
	}
}

func SchemaToJSON(schemaString string) ([]byte, error) {
	s, err := schema.Parse(schemaString)
	if err != nil {
		return nil, err
	}

	result, err2 := exec.IntrospectSchema(s)
	if err2 != nil {
		return nil, err
	}

	return json.Marshal(result)
}

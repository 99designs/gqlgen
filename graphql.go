package graphql

import (
	"context"
	"encoding/json"

	opentracing "github.com/opentracing/opentracing-go"
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
	document, err := query.Parse(queryString)
	if err != nil {
		return &Response{
			Errors: []*errors.GraphQLError{err},
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

	data, errs := exec.ExecuteRequest(subCtx, s.exec, document, operationName, variables)
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

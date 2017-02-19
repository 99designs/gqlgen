package graphql

import (
	"context"
	"encoding/json"
	"fmt"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/neelance/graphql-go/errors"
	"github.com/neelance/graphql-go/internal/exec"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
	"github.com/neelance/graphql-go/introspection"
)

const OpenTracingTagQuery = "graphql.query"
const OpenTracingTagOperationName = "graphql.operationName"
const OpenTracingTagVariables = "graphql.variables"

const OpenTracingTagType = "graphql.type"
const OpenTracingTagField = "graphql.field"
const OpenTracingTagTrivial = "graphql.trivial"
const OpenTracingTagArgsPrefix = "graphql.args."
const OpenTracingTagError = "graphql.error"

type ID string

func (_ ID) ImplementsGraphQLType(name string) bool {
	return name == "ID"
}

func (id *ID) UnmarshalGraphQL(input interface{}) error {
	switch input := input.(type) {
	case string:
		*id = ID(input)
		return nil
	default:
		return fmt.Errorf("wrong type")
	}
}

func ParseSchema(schemaString string, resolver interface{}) (*Schema, error) {
	s := &Schema{
		schema:         schema.New(),
		MaxParallelism: 10,
	}
	if err := s.schema.Parse(schemaString); err != nil {
		return nil, err
	}

	if resolver != nil {
		e, err := exec.Make(s.schema, resolver)
		if err != nil {
			return nil, err
		}
		s.exec = e
	}

	return s, nil
}

func MustParseSchema(schemaString string, resolver interface{}) *Schema {
	s, err := ParseSchema(schemaString, resolver)
	if err != nil {
		panic(err)
	}
	return s
}

type Schema struct {
	schema *schema.Schema
	exec   *exec.Exec

	// MaxParallelism specifies the maximum number of resolvers per request allowed to run in parallel.
	MaxParallelism int
}

type Response struct {
	Data       interface{}            `json:"data,omitempty"`
	Errors     []*errors.QueryError   `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

func (s *Schema) Exec(ctx context.Context, queryString string, operationName string, variables map[string]interface{}) *Response {
	if s.exec == nil {
		panic("schema created without resolver, can not exec")
	}

	document, err := query.Parse(queryString, s.schema.Resolve)
	if err != nil {
		return &Response{
			Errors: []*errors.QueryError{err},
		}
	}

	span, subCtx := opentracing.StartSpanFromContext(ctx, "GraphQL request")
	span.SetTag(OpenTracingTagQuery, queryString)
	if operationName != "" {
		span.SetTag(OpenTracingTagOperationName, operationName)
	}
	if len(variables) != 0 {
		span.SetTag(OpenTracingTagVariables, variables)
	}
	defer span.Finish()

	data, errs := exec.ExecuteRequest(subCtx, s.exec, document, operationName, variables, s.MaxParallelism)
	if len(errs) != 0 {
		ext.Error.Set(span, true)
		span.SetTag(OpenTracingTagError, errs)
	}
	return &Response{
		Data:   data,
		Errors: errs,
	}
}

func (s *Schema) Inspect() *introspection.Schema {
	return introspection.WrapSchema(s.schema)
}

func (s *Schema) ToJSON() ([]byte, error) {
	result, err := exec.IntrospectSchema(s.schema)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(result, "", "\t")
}

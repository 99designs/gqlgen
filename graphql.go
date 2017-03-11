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

// ID represents GraphQL's "ID" type. A custom type may be used instead.
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

// ParseSchema parses a GraphQL schema and attaches the given root resolver. It returns an error if
// the Go type signature of the resolvers does not match the schema. If nil is passed as the
// resolver, then the schema can not be executed, but it may be inspected (e.g. with ToJSON).
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

// MustParseSchema calls ParseSchema and panics on error.
func MustParseSchema(schemaString string, resolver interface{}) *Schema {
	s, err := ParseSchema(schemaString, resolver)
	if err != nil {
		panic(err)
	}
	return s
}

// Schema represents a GraphQL schema with an optional resolver.
type Schema struct {
	schema *schema.Schema
	exec   *exec.Exec

	// MaxParallelism specifies the maximum number of resolvers per request allowed to run in parallel. The default is 10.
	MaxParallelism int
}

// Response represents a typical response of a GraphQL server. It may be encoded to JSON directly or
// it may be further processed to a custom response type, for example to include custom error data.
type Response struct {
	Data       interface{}            `json:"data,omitempty"`
	Errors     []*errors.QueryError   `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// Exec executes the given query with the schema's resolver. It panics if the schema was created
// without a resolver. If the context get cancelled, no further resolvers will be called and a
// the context error will be returned as soon as possible (not immediately).
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

// Inspect allows inspection of the given schema.
func (s *Schema) Inspect() *introspection.Schema {
	return introspection.WrapSchema(s.schema)
}

// ToJSON encodes the schema in a JSON format used by tools like Relay.
func (s *Schema) ToJSON() ([]byte, error) {
	result, err := exec.IntrospectSchema(s.schema)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(result, "", "\t")
}

package exec

import (
	"bytes"
	"fmt"

	"github.com/vektah/graphql-go/errors"
	"github.com/vektah/graphql-go/introspection"
	"github.com/vektah/graphql-go/jsonw"
	"github.com/vektah/graphql-go/query"
	"github.com/vektah/graphql-go/schema"
	"github.com/vektah/graphql-go/validation"
)

type ExecutionContext struct {
	variables map[string]interface{}
	errors    []*errors.QueryError
	schema    *schema.Schema
}

type Root interface {
	Query(ec *ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable
	Mutation(ec *ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable
}

type ResolverFunc func(ec *ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable

func (c *ExecutionContext) Errorf(format string, args ...interface{}) {
	c.errors = append(c.errors, errors.Errorf(format, args...))
}

func (c *ExecutionContext) Error(err error) {
	c.errors = append(c.errors, errors.Errorf("%s", err.Error()))
}

func (c *ExecutionContext) IntrospectSchema() *introspection.Schema {
	return introspection.WrapSchema(c.schema)
}

func (c *ExecutionContext) IntrospectType(name string) *introspection.Type {
	t := c.schema.Resolve(name)
	if t == nil {
		return nil
	}
	return introspection.WrapType(t)
}

func getOperation(document *query.Document, operationName string) (*query.Operation, error) {
	if len(document.Operations) == 0 {
		return nil, fmt.Errorf("no operations in query document")
	}

	if operationName == "" {
		if len(document.Operations) > 1 {
			return nil, fmt.Errorf("more than one operation in query document and no operation name given")
		}
		for _, op := range document.Operations {
			return op, nil // return the one and only operation
		}
	}

	op := document.Operations.Get(operationName)
	if op == nil {
		return nil, fmt.Errorf("no operation with name %q", operationName)
	}
	return op, nil
}

func ExecuteRequest(root Root, schema *schema.Schema, document string, operationName string, variables map[string]interface{}) *jsonw.Response {
	doc, qErr := query.Parse(document)
	if qErr != nil {
		return &jsonw.Response{Errors: []*errors.QueryError{qErr}}
	}

	errs := validation.Validate(schema, doc)
	if len(errs) != 0 {
		return &jsonw.Response{Errors: errs}
	}

	op, err := getOperation(doc, operationName)
	if err != nil {
		return &jsonw.Response{Errors: []*errors.QueryError{errors.Errorf("%s", err)}}
	}

	// TODO: variable coercion?

	c := ExecutionContext{
		variables: variables,
		schema:    schema,
	}

	var rootType ResolverFunc

	if op.Type == query.Query {
		rootType = root.Query
	} else if op.Type == query.Mutation {
		rootType = root.Mutation
	} else {
		return &jsonw.Response{Errors: []*errors.QueryError{errors.Errorf("unsupported operation type")}}
	}

	// TODO: parallelize if query.
	data := c.ExecuteSelectionSet(op.Selections, rootType, true)
	b := &bytes.Buffer{}
	data.JSON(b)
	return &jsonw.Response{
		Data:   b.Bytes(),
		Errors: c.errors,
	}
}

func (c *ExecutionContext) ExecuteSelectionSet(sel []query.Selection, resolver ResolverFunc, objectValue interface{}) jsonw.Encodable {
	if objectValue == nil {
		return jsonw.Null
	}
	groupedFieldSet := c.collectFields(sel, map[string]interface{}{})
	fmt.Println("ESS grouped selections")
	for _, s := range groupedFieldSet {
		fmt.Println(s.Alias)
	}
	resultMap := jsonw.Map{}

	for _, collectedField := range groupedFieldSet {
		resultMap.Set(collectedField.Alias, resolver(c, objectValue, collectedField.Name, collectedField.Args, collectedField.Selections))
	}
	return resultMap
}

type CollectedField struct {
	Alias      string
	Name       string
	Args       map[string]interface{}
	Selections []query.Selection
}

func findField(c *[]CollectedField, field *query.Field, vars map[string]interface{}) *CollectedField {
	for i, cf := range *c {
		if cf.Alias == field.Alias.Name {
			return &(*c)[i]
		}
	}

	f := CollectedField{
		Alias: field.Alias.Name,
		Name:  field.Name.Name,
	}
	if len(field.Arguments) > 0 {
		f.Args = map[string]interface{}{}
		for _, arg := range field.Arguments {
			f.Args[arg.Name.Name] = arg.Value.Value(vars)
		}
	}

	*c = append(*c, f)
	return &(*c)[len(*c)-1]
}

func (c *ExecutionContext) collectFields(selSet []query.Selection, visited map[string]interface{}) []CollectedField {
	var groupedFields []CollectedField

	// TODO: Basically everything.
	for _, sel := range selSet {
		switch sel := sel.(type) {
		case *query.Field:
			f := findField(&groupedFields, sel, c.variables)
			f.Selections = append(f.Selections, sel.Selections...)
		default:
			panic(fmt.Errorf("unsupported %T", sel))
		}
	}

	return groupedFields
}

package todoresolver

import (
	"bytes"
	"fmt"

	"github.com/vektah/graphql-go/errors"
	"github.com/vektah/graphql-go/internal/query"
	"github.com/vektah/graphql-go/internal/validation"
	"github.com/vektah/graphql-go/jsonw"
)

type ExecutionContext struct {
	variables map[string]interface{}
	errors    []*errors.QueryError
	resolvers Resolvers
}

type Type interface {
	GetField(field string) Type
	Execute(ec *ExecutionContext, object interface{}, field string, arguments map[string]interface{}, sels []query.Selection) jsonw.Encodable
}

func (c *ExecutionContext) errorf(format string, args ...interface{}) {
	c.errors = append(c.errors, errors.Errorf(format, args...))
}

func (c *ExecutionContext) error(err error) {
	c.errors = append(c.errors, errors.Errorf("%s", err.Error()))
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

func ExecuteRequest(resolvers Resolvers, document string, operationName string, variables map[string]interface{}) *jsonw.Response {
	doc, qErr := query.Parse(document)
	if qErr != nil {
		return &jsonw.Response{Errors: []*errors.QueryError{qErr}}
	}

	errs := validation.Validate(parsedSchema, doc)
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
		resolvers: resolvers,
	}

	var rootType Type = queryType{}

	if op.Type == query.Mutation {
		rootType = mutationType{}
	}

	// TODO: parallelize if query.
	data := c.executeSelectionSet(op.Selections, rootType, nil)
	b := &bytes.Buffer{}
	data.JSON(b)
	return &jsonw.Response{
		Data:   b.Bytes(),
		Errors: c.errors,
	}
}

func (c *ExecutionContext) executeSelectionSet(sel []query.Selection, objectType Type, objectValue interface{}) jsonw.Encodable {
	groupedFieldSet := c.collectFields(objectType, sel, map[string]interface{}{})
	fmt.Println("ESS grouped selections")
	for _, s := range groupedFieldSet {
		fmt.Println(s.Alias)
	}
	resultMap := jsonw.Map{}

	for _, collectedField := range groupedFieldSet {
		//fieldType := objectType.GetField(collectedField.Name)
		//if fieldType == nil {
		//	continue
		//}
		resultMap.Set(collectedField.Alias, objectType.Execute(c, objectValue, collectedField.Name, map[string]interface{}{}, collectedField.Selections))
	}
	return resultMap
}

type CollectedField struct {
	Alias      string
	Name       string
	Selections []query.Selection
}

func findField(c *[]CollectedField, alias string, name string) *CollectedField {
	for i, cf := range *c {
		if cf.Alias == alias {
			return &(*c)[i]
		}
	}

	*c = append(*c, CollectedField{Alias: alias, Name: name})
	return &(*c)[len(*c)-1]
}

func (c *ExecutionContext) collectFields(objectType Type, selSet []query.Selection, visited map[string]interface{}) []CollectedField {
	var groupedFields []CollectedField

	// TODO: Basically everything.
	for _, sel := range selSet {
		switch sel := sel.(type) {
		case *query.Field:
			f := findField(&groupedFields, sel.Alias.Name, sel.Name.Name)
			f.Selections = append(f.Selections, sel.Selections...)
		default:
			panic("Unsupported!")
		}
	}

	return groupedFields
}

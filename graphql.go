package graphql

import (
	"encoding/json"
	"fmt"

	"github.com/neelance/graphql-go/internal/exec"
	"github.com/neelance/graphql-go/internal/query"
	"github.com/neelance/graphql-go/internal/schema"
)

type Schema struct {
	exec *exec.Exec
}

func NewSchema(schemaString string, filename string, resolver interface{}) (*Schema, error) {
	s, err := schema.Parse(schemaString, filename)
	if err != nil {
		return nil, err
	}

	return &Schema{
		exec: exec.Make(s, resolver),
	}, nil
}

func (s *Schema) Exec(queryString string, operationName string, variables map[string]interface{}) ([]byte, error) {
	d, err := query.Parse(queryString)
	if err != nil {
		return nil, err
	}

	if operationName == "" && len(d.Operations) == 1 {
		for name := range d.Operations {
			operationName = name
		}
	}

	op, ok := d.Operations[operationName]
	if !ok {
		return nil, fmt.Errorf("no operation with name %q", operationName)
	}

	rawRes := s.exec.Exec(d, variables, op.SelSet)
	return json.Marshal(rawRes)
}

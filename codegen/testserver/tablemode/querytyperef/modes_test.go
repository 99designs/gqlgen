package querytyperef_test

import (
	"testing"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/modes"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/querytyperef/functions"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/querytyperef/table"
)

// TestModes compares the modes on a field that returns a type called Query, which is not
// the query type.
func TestModes(t *testing.T) {
	modes.Compare(t, functions.NewSchema, table.NewSchema, `mutation { ping }`, `ping`)
	modes.Compare(t, functions.NewSchema, table.NewSchema, `mutation { q { __typename } }`, `Query`)
}

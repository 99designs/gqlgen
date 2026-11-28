package badlinked_test

import (
	"testing"

	"github.com/99designs/gqlgen/codegen/testserver/tablemode/badlinked/functions"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/badlinked/table"
	"github.com/99designs/gqlgen/codegen/testserver/tablemode/modes"
)

// TestModes compares the modes on a schema that a plugin changed with directives where
// their definitions do not allow them.
func TestModes(t *testing.T) {
	for _, tc := range []struct{ query, contains string }{
		{`{ pluginMisplaced }`, `misplaced`},
		{`{ pluginObjectOnField }`, `pluginObjectOnField`},
		{`{ pluginInputObjectOnField }`, `pluginInputObjectOnField`},
		{`{ pluginTyped }`, `pluginTyped`},
		{`{ pluginInput(in: { v: "x" }) }`, `pluginInput`},
	} {
		t.Run(tc.query, func(t *testing.T) {
			modes.Compare(t, functions.NewSchema, table.NewSchema, tc.query, tc.contains)
		})
	}
}

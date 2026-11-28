//go:build !race && gqlgen_table

package tablemode_test

import "testing"

// execMode is the exec mode that the generator of the test binary generates in: with the
// gqlgen_table tag, table mode.
const execMode = "table"

// requireTableMode lets the test, which checks what only table mode generates, run.
func requireTableMode(*testing.T) {}

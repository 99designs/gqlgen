//go:build !race && !gqlgen_table

package tablemode_test

import "testing"

// execMode is the exec mode that the generator of the test binary generates in: without
// the gqlgen_table tag, the functions mode.
const execMode = "functions"

// requireTableMode skips the test, which checks what only table mode generates.
func requireTableMode(t *testing.T) {
	t.Helper()
	t.Skip("table mode needs the gqlgen_table tag")
}

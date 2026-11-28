// Package data holds a Go type of TestTableModeCompiles in a package whose name the
// generated code of table mode used for a variable.
package data

import "github.com/99designs/gqlgen/codegen/testserver/tablemode/compile/field"

// Filter is an input whose field q is set by a resolver.
type Filter struct {
	Q     *string
	Range *field.Range
}

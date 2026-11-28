// Package field holds a Go type of TestTableModeCompiles in a package whose name the
// generated code of table mode used for a parameter.
package field

// Range is an input.
type Range struct {
	From *int
	To   *int
}

// Thing is an object that the schema reaches only through an interface.
type Thing struct {
	ID *int
}

func (*Thing) IsNode() {}

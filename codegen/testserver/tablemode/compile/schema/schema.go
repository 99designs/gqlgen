// Package schema holds a Go type of TestTableModeCompiles in a package whose name the
// generated code of table mode used for a parameter.
package schema

import (
	"fmt"
	"io"
	"strconv"
)

// Level marshals and unmarshals itself.
type Level string

func (l Level) MarshalGQL(w io.Writer) { _, _ = io.WriteString(w, strconv.Quote(string(l))) }

func (l *Level) UnmarshalGQL(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("%T is not a string", v)
	}
	*l = Level(s)
	return nil
}

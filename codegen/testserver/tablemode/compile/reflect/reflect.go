// Package reflect holds a Go type of TestTableModeCompiles in a package whose name the
// generated code of table mode used for a parameter or an import.
package reflect

import (
	"fmt"
	"io"
	"strconv"
)

// Mirror marshals and unmarshals itself.
type Mirror string

func (g Mirror) MarshalGQL(w io.Writer) { _, _ = io.WriteString(w, strconv.Quote(string(g))) }

func (g *Mirror) UnmarshalGQL(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("%T is not a string", v)
	}
	*g = Mirror(s)
	return nil
}

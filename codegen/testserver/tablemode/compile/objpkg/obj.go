// Package obj holds a Go type of TestTableModeCompiles in a package whose name the
// generated code of table mode used for a parameter. Its directory has another name, as
// obj/ is often ignored.
package obj

import (
	"fmt"
	"io"
	"strconv"
)

// Grade marshals and unmarshals itself.
type Grade string

func (g Grade) MarshalGQL(w io.Writer) { _, _ = io.WriteString(w, strconv.Quote(string(g))) }

func (g *Grade) UnmarshalGQL(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("%T is not a string", v)
	}
	*g = Grade(s)
	return nil
}

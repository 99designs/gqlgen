// Package v holds a scalar whose package has the name of the parameter of the unmarshalers
// of the functions mode, which hides the package in their bodies.
package v

import "io"

type Scalar int

func (s Scalar) MarshalGQL(w io.Writer) {}

func (s *Scalar) UnmarshalGQL(v any) error { return nil }

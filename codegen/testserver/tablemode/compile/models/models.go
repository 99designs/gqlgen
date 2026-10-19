// Package models holds the Go types that the schemas of TestTableModeCompiles bind to.
package models

import (
	"io"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

// OmitItem binds a field of an object to an Omittable, which the binder unwraps: the
// functions mode returns the Omittable, which marshals itself.
type OmitItem struct {
	Name graphql.Omittable[*string]
}

// MapItem binds a field whose type is modeled as map[string]any to a map[string]string,
// which the functions mode rejects when it resolves the field.
type MapItem struct {
	Extra map[string]string
}

// RawInput binds an input that marshals and unmarshals itself, whose fields neither mode
// reads.
type RawInput struct {
	A *int
}

func (r *RawInput) UnmarshalGQL(v any) error { return nil }

func (r RawInput) MarshalGQL(w io.Writer) {}

// UnmarshalOnly binds an input that unmarshals itself but does not marshal itself, which
// the functions mode does not compile.
type UnmarshalOnly struct {
	A *int
}

func (u *UnmarshalOnly) UnmarshalGQL(v any) error { return nil }

// Node is the Go type of an interface whose other models are NamedNode, another
// interface, and Member, which implements it.
type Node interface{ IsNode() }

type NamedNode interface {
	Node
	Name() string
}

type Member struct{ ID *int }

func (*Member) IsNode() {}

type Holder struct {
	Item  *Member
	Named NamedNode
}

// Tags is a map that implements Node by a value receiver, which the functions mode passes
// to its marshaler by its address.
type Tags map[string]string

func (Tags) IsNode() {}

// Account and AccountView are the Go types of one object, and Filter and FilterView those
// of one input. Viewer returns and takes the second ones.
type (
	Account     struct{ ID *int }
	AccountView struct{ ID *int }
	Filter      struct{ ID *int }
	FilterView  struct{ ID *int }
)

type Viewer struct {
	Account  *AccountView
	Accounts []*AccountView
}

func (Viewer) Search(filter *FilterView, filters [][]*FilterView) *int { return nil }

// MarshalDate and UnmarshalDate bind a scalar to time.Time. MarshalDate returns a
// graphql.WriterFunc, which the functions mode returns as a graphql.Marshaler where the
// scalar is nullable.
func MarshalDate(t time.Time) graphql.WriterFunc {
	return func(w io.Writer) { _, _ = io.WriteString(w, `"`+t.Format(time.DateOnly)+`"`) }
}

func UnmarshalDate(v any) (time.Time, error) {
	s, _ := v.(string)
	return time.Parse(time.DateOnly, s)
}

// PtrMarshal marshals itself with a pointer receiver, so that a value of it is not a
// graphql.Marshaler, which the functions mode does not compile.
type PtrMarshal int

func (p *PtrMarshal) MarshalGQL(w io.Writer) {}

func (p *PtrMarshal) UnmarshalGQL(v any) error { return nil }

// BadUnmarshal unmarshals itself from a string, so that the functions mode, which passes
// it an any, does not compile.
type BadUnmarshal int

func (b BadUnmarshal) MarshalGQL(w io.Writer) {}

func (b *BadUnmarshal) UnmarshalGQL(v string) error { return nil }

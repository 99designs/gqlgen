package codegen

import "strings"

type NamedTypes map[string]*NamedType

type NamedType struct {
	IsScalar    bool
	IsInterface bool
	GQLType     string // Name of the graphql type
	GoType      string // Name of the go type
	Package     string // the package the go type lives in
	Import      *Import
}

type Type struct {
	*NamedType

	Modifiers []string
}

const (
	modList = "[]"
	modPtr  = "*"
)

func (t NamedType) FullName() string {
	if t.Import == nil || t.Import.Name == "" {
		return t.GoType
	}
	return t.Import.Name + "." + t.GoType
}

func (t Type) Signature() string {
	return strings.Join(t.Modifiers, "") + t.FullName()
}

func (t Type) IsPtr() bool {
	return len(t.Modifiers) > 0 && t.Modifiers[0] == modPtr
}

func (t Type) IsSlice() bool {
	return len(t.Modifiers) > 0 && t.Modifiers[0] == modList
}

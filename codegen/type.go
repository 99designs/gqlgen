package codegen

import (
	"strings"
)

type NamedTypes map[string]*NamedType

type NamedType struct {
	Ref
	IsScalar    bool
	IsInterface bool
	GQLType     string // Name of the graphql type
	Marshaler   *Ref   // If this type has an external marshaler this will be set
}

type Ref struct {
	GoType  string  // Name of the go type
	Package string  // the package the go type lives in
	Import  *Import // the resolved import with alias
}

type Type struct {
	*NamedType

	Modifiers []string
}

const (
	modList = "[]"
	modPtr  = "*"
)

func (t Ref) FullName() string {
	return t.pkgDot() + t.GoType
}

func (t Ref) pkgDot() string {
	if t.Import == nil || t.Import.Name == "" {
		return ""
	}
	return t.Import.Name + "."
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

func (t NamedType) IsMarshaled() bool {
	return t.Marshaler != nil
}

func (t Type) Unmarshal(result, raw string) string {
	if t.Marshaler != nil {
		return result + ", err := " + t.Marshaler.pkgDot() + "Unmarshal" + t.Marshaler.GoType + "(" + raw + ")"
	}
	return tpl(`var {{.result}} {{.type}}
		err := (&{{.result}}).UnmarshalGQL({{.raw}})`, map[string]interface{}{
		"result": result,
		"raw":    raw,
		"type":   t.FullName(),
	})
}

func (t Type) Marshal(result, val string) string {
	if t.Marshaler != nil {
		return result + " = " + t.Marshaler.pkgDot() + "Marshal" + t.Marshaler.GoType + "(" + val + ")"
	}

	return result + " = " + val
}

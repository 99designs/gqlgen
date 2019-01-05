package codegen

import "github.com/99designs/gqlgen/codegen/templates"

type NamedTypes map[string]*TypeDefinition

type TypeDefinition struct {
	TypeImplementation
	IsScalar    bool
	IsInterface bool
	IsInput     bool
	GQLType     string              // Name of the graphql type
	Marshaler   *TypeImplementation // If this type has an external marshaler this will be set
}

type TypeImplementation struct {
	GoType        string // Name of the go type
	Package       string // the package the go type lives in
	IsUserDefined bool   // does the type exist in the typemap
}

const (
	modList = "[]"
	modPtr  = "*"
)

func (t TypeImplementation) FullName() string {
	return t.PkgDot() + t.GoType
}

func (t TypeImplementation) PkgDot() string {
	name := templates.CurrentImports.Lookup(t.Package)
	if name == "" {
		return ""

	}

	return name + "."
}

func (t TypeDefinition) IsMarshaled() bool {
	return t.Marshaler != nil
}

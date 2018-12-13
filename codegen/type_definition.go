package codegen

import (
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/ast"
)

type NamedTypes map[string]*TypeDefinition

// TypeDefinition is the static reference to a graphql type. It can be referenced by many TypeReferences,
// and has one or more backing implementations in go.
type TypeDefinition struct {
	TypeImplementation
	IsScalar    bool
	IsInterface bool
	IsInput     bool
	GQLType     string              // Name of the graphql type
	Marshaler   *TypeImplementation // If this type has an external marshaler this will be set
}

// TypeImplementation is a reference to exisiting golang code that either meets the graphql.Marshaler interface
// or points to the root of a pair of external Marshal[TYPE] and Unmarshal[TYPE] functions.
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

func (n NamedTypes) getType(t *ast.Type) *TypeReference {
	orig := t
	var modifiers []string
	for {
		if t.Elem != nil {
			modifiers = append(modifiers, modList)
			t = t.Elem
		} else {
			if !t.NonNull {
				modifiers = append(modifiers, modPtr)
			}
			if n[t.NamedType] == nil {
				panic("missing type " + t.NamedType)
			}
			res := &TypeReference{
				TypeDefinition: n[t.NamedType],
				Modifiers:      modifiers,
				ASTType:        orig,
			}

			if res.IsInterface {
				res.StripPtr()
			}

			return res
		}
	}
}

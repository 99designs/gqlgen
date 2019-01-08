package codegen

import (
	"go/types"

	"github.com/vektah/gqlparser/ast"
)

type NamedTypes map[string]*TypeDefinition

// TypeDefinition is the static reference to a graphql type. It can be referenced by many TypeReferences,
// and has one or more backing implementations in go.
type TypeDefinition struct {
	IsScalar    bool
	IsInterface bool
	IsInput     bool
	GQLType     string      // Name of the graphql type
	GoType      types.Type  // The backing go type, may be nil until after model generation
	Marshaler   *types.Func // When using external marshalling functions this will point to the Marshal function
	Unmarshaler *types.Func // When using external marshalling functions this will point to the Unmarshal function
}

const (
	modList = "[]"
	modPtr  = "*"
)

func (t TypeDefinition) IsMarshaled() bool {
	return t.Marshaler != nil || t.Unmarshaler != nil
}

func (t TypeDefinition) IsEmptyInterface() bool {
	i, isInterface := t.GoType.(*types.Interface)
	return isInterface && i.NumMethods() == 0
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
				Definition: n[t.NamedType],
				Modifiers:  modifiers,
				ASTType:    orig,
			}

			if res.Definition.IsInterface {
				res.StripPtr()
			}

			return res
		}
	}
}

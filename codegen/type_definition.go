package codegen

import (
	"go/types"

	"github.com/vektah/gqlparser/ast"
)

type NamedTypes map[string]*TypeDefinition

// TypeDefinition is the static reference to a graphql type. It can be referenced by many TypeReferences,
// and has one or more backing implementations in go.
type TypeDefinition struct {
	GQLDefinition *ast.Definition
	GoType        types.Type  // The backing go type, may be nil until after model generation
	Marshaler     *types.Func // When using external marshalling functions this will point to the Marshal function
	Unmarshaler   *types.Func // When using external marshalling functions this will point to the Unmarshal function
}

func (t TypeDefinition) IsMarshaled() bool {
	return t.Marshaler != nil || t.Unmarshaler != nil
}

func (t TypeDefinition) IsEmptyInterface() bool {
	i, isInterface := t.GoType.(*types.Interface)
	return isInterface && i.NumMethods() == 0
}

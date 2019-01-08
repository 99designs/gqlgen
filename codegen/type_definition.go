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

func (t TypeDefinition) IsMarshaled() bool {
	return t.Marshaler != nil || t.Unmarshaler != nil
}

func (t TypeDefinition) IsEmptyInterface() bool {
	i, isInterface := t.GoType.(*types.Interface)
	return isInterface && i.NumMethods() == 0
}

func (n NamedTypes) goTypeForAst(t *ast.Type) types.Type {
	if t.Elem != nil {
		return types.NewSlice(n.goTypeForAst(t.Elem))
	}

	nt := n[t.NamedType]
	gt := nt.GoType
	if gt == nil {
		panic("missing type " + t.NamedType)
	}

	if !t.NonNull && !nt.IsInterface {
		return types.NewPointer(gt)
	}

	return gt
}

func (n NamedTypes) getType(t *ast.Type) *TypeReference {
	return &TypeReference{
		Definition: n[t.Name()],
		GoType:     n.goTypeForAst(t),
		ASTType:    t,
	}
}
